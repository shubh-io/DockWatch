package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ============================================================================
// Container types
// ============================================================================

// Container holds all the data we show in the TUI
type Container struct {
	ID      string   // short container id
	Names   []string // can have multiple names
	Image   string   // image name like "nginx:latest"
	Status  string   // human readable status
	State   string   // running/exited/etc
	Memory  string   // mem usage %
	CPU     string   // cpu usage %
	PIDs    string   // process count
	NetIO   string   // network I/O
	BlockIO string   // block I/O
}

// sent when we finish fetching the container list
type containersMsg struct {
	containers []Container
	err        error
}

// sent when logs are ready
type logsMsg struct {
	id    string
	lines []string
	err   error
}

// ============================================================================
// Docker stats
// ============================================================================

// grab cpu/mem/pids for a container
// returns empty strings on error so we don't block the UI
func GetContainerStats(containerID string) (cpu string, mem string, pids string, NetIO string, BlockIO string, err error) {
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

// ============================================================================
// Logs
// ============================================================================

// fetch logs from a container
// skips empty lines and trims whitespace
func GetLogs(containerID string) ([]string, error) {
	// 5 sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// run docker logs but only tail the last 100 lines to avoid huge output
	// using the CLI --tail is more efficient than fetching everything then truncating
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

// ============================================================================
// List containers
// ============================================================================

// get all containers using docker CLI
// grabs live stats for running ones
func ListContainersUsingCLI() ([]Container, error) {
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

	// docker ps returns json like this
	type psEntry struct {
		ID     string `json:"ID"`
		Names  string `json:"Names"`
		Image  string `json:"Image"`
		Status string `json:"Status"`
		State  string `json:"State"`
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

		// split comma-separated names
		names := []string{}
		if e.Names != "" {
			for _, n := range strings.Split(e.Names, ",") {
				names = append(names, strings.TrimSpace(n))
			}
		}

		// build container struct
		container := Container{
			ID:     e.ID,
			Names:  names,
			Image:  e.Image,
			Status: e.Status,
			State:  e.State,
		}

		// get live stats if container is running
		if e.State == "running" {
			cpu, mem, pids, netio, blockio, err := GetContainerStats(e.ID)
			if err == nil {
				// only set if we got them
				container.CPU = cpu
				container.Memory = mem
				container.PIDs = pids
				container.NetIO = netio
				container.BlockIO = blockio
			}
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

	return out, nil
}

// ============================================================================
// Container actions
// ============================================================================

// run a docker command on a container
// works with start, stop, restart, rm, etc
func dockerAction(action, containerID string) error {
	// 30 sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", action, containerID)
	return cmd.Run()
}
