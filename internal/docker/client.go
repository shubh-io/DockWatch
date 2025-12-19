package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GetContainerStats grabs cpu/mem/pids for a container
// returns empty strings on error so we don't block the UI
func GetContainerStats(containerID string) (cpu string, mem string, pids string, netIO string, blockIO string, err error) {
	// 3 sec timeout because some containers are weird and hang
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// --no-stream = instant snapshot, not continuous
	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{json .}}", containerID)

	output, err := cmd.Output()
	if err != nil {
		// timeout or error, just bail
		return "", "", "", "", "", err
	}

	// docker stats returns json like this
	type statsEntry struct {
		CPUPerc string `json:"CPUPerc"`
		MemPerc string `json:"MemPerc"`
		PIDs    string `json:"PIDs"`
		NetIO   string `json:"NetIO"`
		BlockIO string `json:"BlockIO"`
	}

	// parse it
	var s statsEntry
	if err := json.Unmarshal(output, &s); err != nil {
		return "", "", "", "", "", err
	}

	return s.CPUPerc, s.MemPerc, s.PIDs, s.NetIO, s.BlockIO, nil
}

// GetLogs fetches logs from a container
// skips empty lines and trims whitespace
func GetLogs(containerID string) ([]string, error) {
	// 5 sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// run docker logs but only tail the last 100 lines to avoid huge output
	// using the CLI --tail is more efficient than fetching everything then truncating
	// saves resources and time
	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", "100", containerID)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// read output line by line
	scanner := bufio.NewScanner(stdout)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// grab all non-empty lines
	var out []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip blanks
		if line == "" {
			continue
		}
		out = append(out, line)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		cmd.Wait()
		return nil, err
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return out, nil
}

// ListContainers gets all containers using docker CLI
// grabs live stats for running ones
func ListContainers() ([]Container, error) {
	// 30 sec timeout since we fetch stats for each running container
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// docker ps with json output
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{json .}}", "--all")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// read json lines
	scanner := bufio.NewScanner(stdout)

	var out []Container
	var runningIDs []string // collect running container IDs

	// docker ps returns json like this
	type psEntry struct {
		ID     string `json:"ID"`
		Names  string `json:"Names"`
		Image  string `json:"Image"`
		Status string `json:"Status"`
		// State  string `json:"State"`
		Ports string `json:"Ports"`
	}

	// parse each line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var e psEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			cmd.Wait()
			return nil, fmt.Errorf("parsing docker output: %w", err)
		}

		// split comma separated names
		names := []string{}
		if e.Names != "" {
			for _, n := range strings.Split(e.Names, ",") {
				names = append(names, strings.TrimSpace(n))
			}
		}

		// build container struct
		// derive a short state from Status text (ex- "Up 2 minutes" -> "running")
		st := strings.ToLower(strings.TrimSpace(e.Status))
		state := "unknown"
		if strings.HasPrefix(st, "up") {
			state = "running"
		} else if strings.HasPrefix(st, "paused") || strings.Contains(st, "paused") {
			state = "paused"
		} else if strings.Contains(st, "restarting") {
			state = "restarting"
		} else if strings.HasPrefix(st, "exited") || strings.Contains(st, "exited") || strings.Contains(st, "dead") {
			state = "exited"
		} else if strings.HasPrefix(st, "created") {
			state = "created"
		}

		container := Container{
			ID:     e.ID,
			Names:  names,
			Image:  e.Image,
			Status: e.Status,
			State:  state,
			Ports:  e.Ports,
		}

		// collect running container Ids for batch stats fetch (based on derived State)
		if state == "running" {
			runningIDs = append(runningIDs, e.ID)
		}

		out = append(out, container)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		cmd.Wait()
		return nil, err
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	// Fetch stats for all running containers in ONE call
	if len(runningIDs) > 0 {
		statsMap, err := GetAllContainerStats(runningIDs)
		if err == nil {
			// Apply stats to containers
			for i := range out {
				if stats, ok := statsMap[out[i].ID]; ok {
					out[i].CPU = stats.CPU
					out[i].Memory = stats.Memory
					out[i].NetIO = stats.NetIO
					out[i].BlockIO = stats.BlockIO
				}
			}
		}
	}

	return out, nil
}

// GetAllContainerStats fetches stats for multiple containers in a single docker stats call
// This is MUCH MUCH MUCH faster than previously calling docker stats separately for each container
func GetAllContainerStats(containerIDs []string) (map[string]ContainerStats, error) {
	if len(containerIDs) == 0 {
		return nil, nil
	}

	// 5 sec timeout for batch stats (much faster than individual calls)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build command with all container IDs instead of one by one like old logic flow which resulted in more loading time
	args := []string{"stats", "--no-stream", "--format", "{{json .}}"}
	args = append(args, containerIDs...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Read stats JSON lines
	scanner := bufio.NewScanner(stdout)
	statsMap := make(map[string]ContainerStats)

	type statsEntry struct {
		ID      string `json:"ID"`
		CPUPerc string `json:"CPUPerc"`
		MemPerc string `json:"MemPerc"`
		NetIO   string `json:"NetIO"`
		BlockIO string `json:"BlockIO"`
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var s statsEntry
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue // skip malformed lines
		}

		statsMap[s.ID] = ContainerStats{
			CPU:    s.CPUPerc,
			Memory: s.MemPerc,
			// PIDs:    s.PIDs,
			NetIO:   s.NetIO,
			BlockIO: s.BlockIO,
		}
	}

	if err := scanner.Err(); err != nil {
		cmd.Wait()
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return statsMap, nil
}

// DoAction runs a docker command on a container
// works with start, stop, restart, rm, etc
func DoAction(action, containerID string) error {
	// 30 sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", action, containerID)
	return cmd.Run()
}
