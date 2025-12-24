package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/shubh-io/dockmate/internal/config"
)

// runtimeBin returns the configured container runtime binary name (podman or docker).

func runtimeBin() string {
	cfg, err := config.Load()
	if err != nil {
		return "docker"
	}

	rt := strings.TrimSpace(strings.ToLower(cfg.Runtime.Type))
	if rt == "podman" {
		return "podman"
	}

	return "docker"
}

// GetContainerStats grabs cpu/mem/pids for a container
// returns empty strings on error so we don't block the UI
func GetContainerStats(containerID string) (cpu string, mem string, pids string, netIO string, blockIO string, err error) {
	// 3 sec timeout because some containers are weird and hang
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, runtimeBin(), "stats", "--no-stream", "--format", "{{json .}}", containerID)

	output, err := cmd.Output()
	if err != nil {
		return "", "", "", "", "", err
	}

	type statsEntry struct {
		CPUPerc string `json:"CPUPerc"`
		MemPerc string `json:"MemPerc"`
		PIDs    string `json:"PIDs"`
		NetIO   string `json:"NetIO"`
		BlockIO string `json:"BlockIO"`
	}

	var s statsEntry
	if err := json.Unmarshal(output, &s); err != nil {
		return "", "", "", "", "", err
	}

	return s.CPUPerc, s.MemPerc, s.PIDs, s.NetIO, s.BlockIO, nil
}

func GetLogs(containerID string) ([]string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, runtimeBin(), "logs", "--tail", "100", containerID)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var out []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

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

func ListContainers() ([]Container, error) {
	// 30 sec timeout since we fetch stats for each running container
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runtime := runtimeBin()
	var cmd *exec.Cmd

	// Docker returns newline-delimited JSON
	cmd = exec.CommandContext(ctx, runtime, "ps", "--format", "{{json .}}", "--all")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var out []Container
	var runningIDs []string

	if runtime == "podman" {
		// Podman format - JSON array
		type podmanEntry struct {
			Id     string            `json:"Id"`
			Names  []string          `json:"Names"`
			Image  string            `json:"Image"`
			Status string            `json:"Status"`
			State  string            `json:"State"`
			Labels map[string]string `json:"Labels"`
			Ports  []struct {
				HostPort      int    `json:"host_port"`
				ContainerPort int    `json:"container_port"`
				Protocol      string `json:"protocol"`
			} `json:"Ports"`
		}

		var entries []podmanEntry
		if err := json.Unmarshal(output, &entries); err == nil {

			for _, e := range entries {
				// Format ports like Docker does
				ports := ""
				if len(e.Ports) > 0 {
					var portStrs []string
					for _, p := range e.Ports {
						if p.HostPort > 0 {
							portStrs = append(portStrs, fmt.Sprintf("0.0.0.0:%d->%d/%s", p.HostPort, p.ContainerPort, p.Protocol))
						}
					}
					ports = strings.Join(portStrs, ", ")
				}

				state := strings.ToLower(e.State)

				// check for compose project or quadlet unit
				projectName := e.Labels["io.podman.compose.project"]
				if projectName == "" {
					if unit, ok := e.Labels["PODMAN_SYSTEMD_UNIT"]; ok {
						projectName = strings.TrimSuffix(unit, ".service")
					}
				}

				container := Container{
					ID:             e.Id,
					Names:          e.Names,
					Image:          e.Image,
					Status:         e.Status,
					State:          state,
					Ports:          ports,
					ComposeProject: projectName,
				}

				if state == "running" {
					runningIDs = append(runningIDs, e.Id)
				}

				out = append(out, container)
			}
		} else {
			// Fallback -
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}

				var e podmanEntry
				if err := json.Unmarshal([]byte(line), &e); err != nil {
					continue // skip weird lines
				}

				ports := ""
				if len(e.Ports) > 0 {
					var portStrs []string
					for _, p := range e.Ports {
						if p.HostPort > 0 {
							portStrs = append(portStrs, fmt.Sprintf("0.0.0.0:%d->%d/%s", p.HostPort, p.ContainerPort, p.Protocol))
						}
					}
					ports = strings.Join(portStrs, ", ")
				}

				state := strings.ToLower(e.State)

				// check for compose project or quadlet unit
				projectName := e.Labels["io.podman.compose.project"]
				if projectName == "" {
					if unit, ok := e.Labels["PODMAN_SYSTEMD_UNIT"]; ok {
						projectName = strings.TrimSuffix(unit, ".service")
					}
				}

				container := Container{
					ID:             e.Id,
					Names:          e.Names,
					Image:          e.Image,
					Status:         e.Status,
					State:          state,
					Ports:          ports,
					ComposeProject: projectName,
				}

				if state == "running" {
					runningIDs = append(runningIDs, e.Id)
				}

				out = append(out, container)
			}
			if err := scanner.Err(); err != nil {
				return nil, err
			}
		}
	} else {
		type dockerEntry struct {
			ID     string `json:"ID"`
			Names  string `json:"Names"`
			Image  string `json:"Image"`
			Status string `json:"Status"`
			Ports  string `json:"Ports"`
		}

		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var e dockerEntry
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				return nil, fmt.Errorf("parsing docker output: %w", err)
			}

			names := []string{}
			if e.Names != "" {
				for _, n := range strings.Split(e.Names, ",") {
					names = append(names, strings.TrimSpace(n))
				}
			}

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

			if state == "running" {
				runningIDs = append(runningIDs, e.ID)
			}

			out = append(out, container)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	// Fetch stats for all running containers in ONE call
	if len(runningIDs) > 0 {
		statsMap, err := GetAllContainerStats(runningIDs)
		if err == nil {
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

func GetAllContainerStats(containerIDs []string) (map[string]ContainerStats, error) {
	if len(containerIDs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runtime := runtimeBin()

	args := []string{"stats", "--no-stream", "--format"}

	if runtime == "podman" {
		args = append(args, `{"ID":"{{.ID}}","CPUPerc":"{{.CPUPerc}}","MemPerc":"{{.MemPerc}}","NetIO":"{{.NetIO}}","BlockIO":"{{.BlockIO}}"}`)
	} else {
		// for docker
		args = append(args, "{{json .}}")
	}

	args = append(args, containerIDs...)

	cmd := exec.CommandContext(ctx, runtime, args...)
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
			continue // skip weird lines
		}

		mapID := s.ID
		if runtime == "podman" {
			for _, longID := range containerIDs {
				if strings.HasPrefix(longID, s.ID) {
					mapID = longID
					break
				}
			}
		}

		statsMap[mapID] = ContainerStats{
			ID:      mapID,
			CPU:     s.CPUPerc,
			Memory:  s.MemPerc,
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

func DoAction(action, containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, runtimeBin(), action, containerID)
	return cmd.Run()
}

// FetchComposeProjects fetches all Docker/Podman Compose projects with their containers

func FetchComposeProjects() (map[string]*ComposeProject, error) {
	// 30 sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runtime := runtimeBin()
	var cmd *exec.Cmd

	if runtime == "podman" {
		// well podman uses io.podman.compose labels
		cmd = exec.CommandContext(ctx, runtime, "ps", "-a",
			"--filter", "label=io.podman.compose.project",
			"--format", "json")
	} else {
		// and docker uses com.docker.compose labels
		cmd = exec.CommandContext(ctx, runtime, "ps", "-a",
			"--filter", "label=com.docker.compose.project",
			"--format", "{{json .}}")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	projects := make(map[string]*ComposeProject)
	var runningIDs []string

	if runtime == "podman" {
		// Podman format - json array
		type podmanEntry struct {
			Id     string            `json:"Id"`
			Names  []string          `json:"Names"`
			Image  string            `json:"Image"`
			Status string            `json:"Status"`
			State  string            `json:"State"`
			Labels map[string]string `json:"Labels"`
			Ports  []struct {
				HostPort      int    `json:"host_port"`
				ContainerPort int    `json:"container_port"`
				Protocol      string `json:"protocol"`
			} `json:"Ports"`
		}

		var entries []podmanEntry
		if err := json.Unmarshal(output, &entries); err != nil {
			return nil, fmt.Errorf("parsing podman compose output: %w", err)
		}

		for _, e := range entries {

			projectName := e.Labels["io.podman.compose.project"]

			if projectName == "" {
				if unit, ok := e.Labels["PODMAN_SYSTEMD_UNIT"]; ok {
					//removing the .service suffix
					projectName = strings.TrimSuffix(unit, ".service")
				}
			}

			serviceName := e.Labels["io.podman.compose.service"]
			containerNumber := e.Labels["io.podman.compose.container-number"]
			configFile := e.Labels["io.podman.compose.project.config_files"]
			workingDir := e.Labels["io.podman.compose.project.working_dir"]

			if projectName == "" {
				continue
			}

			ports := ""
			if len(e.Ports) > 0 {
				var portStrs []string
				for _, p := range e.Ports {
					if p.HostPort > 0 {
						portStrs = append(portStrs, fmt.Sprintf("0.0.0.0:%d->%d/%s", p.HostPort, p.ContainerPort, p.Protocol))
					}
				}
				ports = strings.Join(portStrs, ", ")
			}

			state := strings.ToLower(e.State)

			container := Container{
				ID:             e.Id,
				Names:          e.Names,
				Image:          e.Image,
				Status:         e.Status,
				State:          state,
				Ports:          ports,
				ComposeProject: projectName,
				ComposeService: serviceName,
				ComposeNumber:  containerNumber,
			}

			if state == "running" {
				runningIDs = append(runningIDs, e.Id)
			}

			project, exists := projects[projectName]
			if !exists {
				project = &ComposeProject{
					Name:       projectName,
					Containers: []Container{},
					ConfigFile: configFile,
					WorkingDir: workingDir,
				}
				projects[projectName] = project
			}

			project.Containers = append(project.Containers, container)
		}
	} else {
		type dockerEntry struct {
			ID        string `json:"ID"`
			Names     string `json:"Names"`
			Image     string `json:"Image"`
			Status    string `json:"Status"`
			State     string `json:"State"`
			Ports     string `json:"Ports"`
			Labels    string `json:"Labels"`
			CreatedAt string `json:"CreatedAt"`
		}

		scanner := bufio.NewScanner(strings.NewReader(string(output)))

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var e dockerEntry
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				continue // Skip malformed entries
			}

			labels := parseLabels(e.Labels)

			projectName := labels["com.docker.compose.project"]
			serviceName := labels["com.docker.compose.service"]
			containerNumber := labels["com.docker.compose.container-number"]

			if projectName == "" {
				continue
			}

			names := []string{}
			if e.Names != "" {
				for _, n := range strings.Split(e.Names, ",") {
					names = append(names, strings.TrimSpace(n))
				}
			}

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
				ID:             e.ID,
				Names:          names,
				Image:          e.Image,
				Status:         e.Status,
				State:          state,
				Ports:          e.Ports,
				ComposeProject: projectName,
				ComposeService: serviceName,
				ComposeNumber:  containerNumber,
			}

			if state == "running" {
				runningIDs = append(runningIDs, e.ID)
			}

			// Get or create project
			project, exists := projects[projectName]
			if !exists {
				project = &ComposeProject{
					Name:       projectName,
					Containers: []Container{},
					ConfigFile: labels["com.docker.compose.project.config_files"],
					WorkingDir: labels["com.docker.compose.project.working_dir"],
				}
				projects[projectName] = project
			}

			// Add container to project
			project.Containers = append(project.Containers, container)
		}

		// Check scanner errors
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	if len(runningIDs) > 0 {
		statsMap, err := GetAllContainerStats(runningIDs)
		if err == nil {

			for _, project := range projects {
				for i := range project.Containers {
					if stats, ok := statsMap[project.Containers[i].ID]; ok {
						project.Containers[i].CPU = stats.CPU
						project.Containers[i].Memory = stats.Memory
						project.Containers[i].NetIO = stats.NetIO
						project.Containers[i].BlockIO = stats.BlockIO
					}
				}
			}
		}
	}

	// Calculate project status
	for _, project := range projects {
		running := 0
		total := len(project.Containers)
		for _, c := range project.Containers {
			if strings.ToLower(c.State) == "running" {
				running++
			}
		}

		if running == total {
			project.Status = AllRunning
		} else if running == 0 {
			project.Status = AllStopped
		} else {
			project.Status = SomeStopped
		}
	}

	return projects, nil
}

// Handles edge cases like commas in values and empty strings
func parseLabels(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	parts := strings.Split(labelsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		idx := strings.Index(part, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])
		labels[key] = value
	}

	return labels
}
