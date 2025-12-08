package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ============================================================================
// PreCheck Types
// ============================================================================

type PreCheckResult struct {
	Passed          bool
	ErrorType       PreCheckErrorType
	ErrorMessage    string
	SuggestedAction string
}

type PreCheckErrorType int

const (
	NoError PreCheckErrorType = iota
	DockerNotInstalled
	DockerDaemonNotRunning
	DockerPermissionDenied
	DockerGroupNotRefreshed
)

// ============================================================================
// PreCheck Functions
// ============================================================================

// checks if the 'docker' group exists on the system and anchor before docker to help find group that 'starts with' docker
func doesDockerGroupExist() bool {
	cmd := exec.Command("grep", "^docker:", "/etc/group")
	err := cmd.Run()
	return err == nil
}

// checks if the current user is listed in the 'docker' group in /etc/group
func isUserInDockerGroup() (bool, error) {
	currentUser := os.Getenv("USER")
	cmd := exec.Command("grep", "^docker:", "/etc/group")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	// output format: docker:x:999:user1,user2,..
	line := string(output)
	parts := strings.Split(line, ":")
	if len(parts) < 4 {
		return false, nil
	}

	// removes whitespaces and 'docker:x:999:' to get only usersInGroup
	usersInGroup := strings.TrimSpace(parts[3])
	if usersInGroup == "" {
		return false, nil
	}
	// split users by comma and check for current user
	users := strings.Split(usersInGroup, ",")
	for _, user := range users {
		if strings.TrimSpace(user) == currentUser {
			return true, nil
		}
	}
	return false, nil
}

// checks if the 'docker' group is in the user's active groups (id -nG)
func isDockerInActiveGroups() (bool, error) {
	cmd := exec.Command("id", "-nG")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	groups := strings.Fields(string(output))
	for _, group := range groups {
		if group == "docker" {
			return true, nil
		}
	}
	return false, nil
}

func checkDockerSocketPermissions() (hasAccess bool, errorMsg string) {
	socketPath := "/var/run/docker.sock"

	// check if socket exists
	_, err := os.Stat(socketPath)
	if err != nil {
		return false, "Docker socket not found at /var/run/docker.sock"
	}

	// try to access the socket with read and write flags(os.O_RDWR)
	file, err := os.OpenFile(socketPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			return false, fmt.Sprintf("Socket exists but insufficient permissions: %v", err)
		}
		return false, fmt.Sprintf("Cannot access socket: %v", err)
	}
	//close the file
	file.Close()

	return true, ""
}

// check if docker is installed

func checkDockerInstalled() PreCheckResult {
	_, err := exec.LookPath("docker")
	if err != nil {
		return PreCheckResult{
			Passed:       false,
			ErrorType:    DockerNotInstalled,
			ErrorMessage: "Docker is not installed or not found in PATH",
			SuggestedAction: "Please install Docker to use this application.\n\n" +
				"Installation guide: https://docs.docker.com/engine/install/",
		}
	}
	return PreCheckResult{Passed: true}
}

func checkDockerDaemon() PreCheckResult {
	cmd := exec.Command("docker", "info")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return PreCheckResult{Passed: true}
	}

	stderrOutput := stderr.String()

	// Check daemon status FIRST
	if strings.Contains(stderrOutput, "Is the docker daemon running") ||
		strings.Contains(stderrOutput, "cannot connect to the Docker daemon") ||
		!isDaemonRunning() {
		return PreCheckResult{
			Passed:       false,
			ErrorType:    DockerDaemonNotRunning,
			ErrorMessage: fmt.Sprintf("Docker daemon is not running.\n\nDocker error:\n%s", stderrOutput),
			SuggestedAction: "Start the Docker service:\n\n" +
				"  sudo systemctl start docker\n\n" +
				"Troubleshooting: https://docs.docker.com/config/daemon/troubleshoot/",
		}
	}

	// Check for permission/connection issues
	if strings.Contains(stderrOutput, "permission denied") ||
		strings.Contains(stderrOutput, "dial unix") {

		inGroupFile, _ := isUserInDockerGroup()
		inActiveGroups, _ := isDockerInActiveGroups()

		// check socket permissions specifically
		hasSocketAccess, socketError := checkDockerSocketPermissions()

		// User is in group (both file and active) but still can't access socket
		if inGroupFile && inActiveGroups && !hasSocketAccess {
			return PreCheckResult{
				Passed:    false,
				ErrorType: DockerPermissionDenied,
				ErrorMessage: fmt.Sprintf("You're in the docker group, but the socket has incorrect permissions.\n\n"+
					"Socket error: %s\n\n"+
					"Docker error:\n%s", socketError, stderrOutput),
				SuggestedAction: "Fix the Docker socket permissions:\n\n" +
					"  sudo chown root:docker /var/run/docker.sock\n" +
					"  sudo chmod 660 /var/run/docker.sock\n\n" +
					"Or restart Docker to recreate the socket:\n\n" +
					"  sudo systemctl restart docker\n\n" +
					"Guide: https://docs.docker.com/engine/install/linux-postinstall/",
			}
		}

		if inGroupFile && !inActiveGroups {
			return PreCheckResult{
				Passed:       false,
				ErrorType:    DockerGroupNotRefreshed,
				ErrorMessage: fmt.Sprintf("You're in the docker group but your session hasn't been refreshed.\n\nDocker error:\n%s", stderrOutput),
				SuggestedAction: "Log out and log back in to refresh your group membership.\n\n" +
					"More info: https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user",
			}
		}

		// Check if docker group exists
		if !doesDockerGroupExist() {
			return PreCheckResult{
				Passed:       false,
				ErrorType:    DockerPermissionDenied,
				ErrorMessage: fmt.Sprintf("Cannot communicate with the Docker daemon.\n\nDocker error:\n%s", stderrOutput),
				SuggestedAction: "The 'docker' group doesn't exist. Create it and add your user:\n\n" +
					"  sudo groupadd docker\n" +
					"  sudo usermod -aG docker $USER\n\n" +
					"Then log out and back in.\n\n" +
					"Guide: https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user",
			}
		}

		// Docker group exists, just need to add user
		return PreCheckResult{
			Passed:       false,
			ErrorType:    DockerPermissionDenied,
			ErrorMessage: fmt.Sprintf("Cannot communicate with the Docker daemon.\n\nDocker error:\n%s", stderrOutput),
			SuggestedAction: "Add your user to the 'docker' group:\n\n" +
				"  sudo usermod -aG docker $USER\n\n" +
				"Then log out and back in.\n\n" +
				"Guide: https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user",
		}
	}

	// Fallback for other errors
	return PreCheckResult{
		Passed:       false,
		ErrorType:    DockerDaemonNotRunning,
		ErrorMessage: fmt.Sprintf("Docker error:\n%s", stderrOutput),
		SuggestedAction: "Check Docker installation and try:\n\n" +
			"  sudo systemctl start docker\n\n" +
			"Docker docs: https://docs.docker.com/",
	}
}

// Helper function to check if daemon is actually running
func isDaemonRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "docker")
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output)) == "active"
}

func RunPreChecks() PreCheckResult {
	// Check 1: Is Docker even installed?
	result := checkDockerInstalled()
	if !result.Passed {
		return result
	}

	// Check 2: Can we connect to Docker daemon
	result = checkDockerDaemon()
	if !result.Passed {
		return result
	}

	return PreCheckResult{Passed: true}
}
