package check

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shubh-io/dockmate/internal/config"
	"github.com/shubh-io/dockmate/internal/tui"
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
	PodmanNotInstalled
	PodmanServiceNotRunning
)

func isPrecheckEnabled() bool {
	cfg, err := config.Load()
	if err != nil {
		return true // default to true if error loading config
	}
	return cfg.Runtime.RunPreChecks
}

// Runtime Selection

// checks if the runtime is properly configured

// Returns false if runtime is set to something other than "docker" or "podman" or config just doesn't exists

func isRuntimeConfigured() bool {
	// fixed- it wasn't checking config file's path on start in last release
	cfgPath, err := config.GetConfigPath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return false
	}

	// Load the config (should succeed if it exists)
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	runtimeType := strings.TrimSpace(strings.ToLower(cfg.Runtime.Type))
	return (runtimeType == "docker" || runtimeType == "podman") && runtimeType != "" && runtimeType != "auto"
}

// promptRuntimeSelection shows the runtime selector TUI and saves selection

func promptRuntimeSelection() error {
	// show runtime selection TUI
	runtimeSelector := tui.NewRuntimeSelectionModel()
	program := tea.NewProgram(runtimeSelector, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("runtime selection failed: %w", err)
	}

	rsModel, ok := finalModel.(tui.RuntimeSelectionModel)
	if !ok {
		return fmt.Errorf("invalid model type returned")
	}

	selectedRuntime := strings.TrimSpace(rsModel.GetChoice())
	if selectedRuntime == "" {
		return fmt.Errorf("no runtime selected")
	}

	// Load current config and update runtime
	cfg, _ := config.Load()
	cfg.Runtime.Type = selectedRuntime

	// Save updated config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save runtime selection: %w", err)
	}

	return nil
}

// ============================================================================
// PreCheck Functions
// ============================================================================

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// getDockerStartCommand detects the init system and returns the appropriate command
func getDockerStartCommand() string {
	if runtime.GOOS == "darwin" {
		return "Start Docker Desktop application"
	}

	// Check for different init systems
	if commandExists("systemctl") {
		return "sudo systemctl start docker"
	}
	if commandExists("rc-service") {
		return "sudo rc-service docker start"
	}
	if commandExists("sv") {
		return "sudo sv up docker"
	}

	// Fallback to generic service command
	return "sudo service docker start"
}

// getDockerRestartCommand detects the init system and returns the restart command
func getDockerRestartCommand() string {
	if runtime.GOOS == "darwin" {
		return "Restart Docker Desktop application"
	}

	// check for different init systems
	if commandExists("systemctl") {
		return "sudo systemctl restart docker"
	}
	if commandExists("rc-service") {
		return "sudo rc-service docker restart"
	}
	if commandExists("sv") {
		return "sudo sv restart docker"
	}

	// Fallback
	return "sudo service docker restart"
}

// getPodmanStartCommand returns their start command per platform (peak user case handling lol)

func getPodmanStartCommand() string {
	if runtime.GOOS == "darwin" {
		return "podman machine start"
	}

	if commandExists("systemctl") {
		return "systemctl --user start podman.socket"
	}

	if commandExists("podman") {
		return "podman system service -t 0 &"
	}

	return "podman machine start"
}

// getPodmanErrorMessage provides platform-specific guidance for starting Podman
func getPodmanErrorMessage() string {
	cmd := getPodmanStartCommand()

	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("Podman machine not running.\n\nQuick fix:\n  %s\n\nIf machine doesn't exist:\n  podman machine init\n  podman machine start\n\nHelp: https://docs.podman.io/", cmd)

	case "linux":
		return fmt.Sprintf("Podman not accessible.\n\nTry:\n  %s\n  \nOr check if rootless is set up:\n  podman info\n\nHelp: https://github.com/containers/podman/blob/main/troubleshooting.md", cmd)

	}

	return fmt.Sprintf("Start Podman: %s\nHelp: https://docs.podman.io/", cmd)
}

// checks if the 'docker' group exists on the system and anchor before docker to help find group that 'starts with' docker
// On macOS, Docker Desktop doesn't use groups, so this always returns false
func doesDockerGroupExist() bool {
	if runtime.GOOS == "darwin" {
		return false
	}

	// check /etc/group on Linux/Unix systems
	if !commandExists("grep") {
		// fallback - check if group file exists and contains docker
		data, err := os.ReadFile("/etc/group")
		if err != nil {
			return false
		}
		return strings.Contains(string(data), "\ndocker:") || strings.HasPrefix(string(data), "docker:")
	}

	cmd := exec.Command("grep", "^docker:", "/etc/group")
	err := cmd.Run()
	return err == nil
}

// checks if the current user is listed in the 'docker' group in /etc/group
// On mac-os, Docker Desktop doesn't use groups, so this always returns false
func isUserInDockerGroup() (bool, error) {
	if runtime.GOOS == "darwin" {
		return false, nil
	}

	// get current user in a cross-platform way
	currentUser, err := user.Current()
	if err != nil {
		return false, err
	}
	username := currentUser.Username

	//reading /etc/group directly if grep is not available
	var output []byte
	if commandExists("grep") {
		cmd := exec.Command("grep", "^docker:", "/etc/group")
		output, err = cmd.Output()
		if err != nil {
			return false, err
		}
	} else {
		// Fallback: read /etc/group and find docker line
		data, err := os.ReadFile("/etc/group")
		if err != nil {
			return false, err
		}
		// split into lines and find docker line
		lines := strings.Split(string(data), "\n")

		for _, line := range lines {
			// find the line that starts with 'docker:'
			if strings.HasPrefix(line, "docker:") {
				output = []byte(line)
				break
			}
		}
		if len(output) == 0 {
			return false, nil
		}
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
		if strings.TrimSpace(user) == username {
			return true, nil
		}
	}
	return false, nil
}

// checks if the 'docker' group is in the user's active groups (id -nG)
// On macOS, Docker Desktop doesn't use groups, so this always returns false
func isDockerInActiveGroups() (bool, error) {
	if runtime.GOOS == "darwin" {
		return false, nil
	}

	// Check if id command exists
	if !commandExists("id") {
		return false, nil
	}

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
	if runtime.GOOS == "darwin" {
		// permissions are managed by Docker Desktop, so skip this check
		return true, ""
	}

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

// check if podman is installed in PATH
func checkPodmanInstalled() PreCheckResult {
	_, err := exec.LookPath("podman")
	if err != nil {
		return PreCheckResult{
			Passed:       false,
			ErrorType:    PodmanNotInstalled,
			ErrorMessage: "Podman is not installed or not found in PATH",
			SuggestedAction: "Please install Podman to use this runtime.\n\n" +
				"Installation guide: https://podman.io/docs/installation",
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
			SuggestedAction: fmt.Sprintf("Start the Docker service:\n\n"+
				"  %s\n\n"+
				"Troubleshooting: https://docs.docker.com/config/daemon/troubleshoot/", getDockerStartCommand()),
		}
	}

	// Check for permission/connection issues
	if strings.Contains(stderrOutput, "permission denied") ||
		strings.Contains(stderrOutput, "dial unix") {

		// macOS Docker Desktop handles permissions differently
		if runtime.GOOS == "darwin" {
			return PreCheckResult{
				Passed:       false,
				ErrorType:    DockerPermissionDenied,
				ErrorMessage: fmt.Sprintf("Cannot connect to Docker Desktop.\n\nDocker error:\n%s", stderrOutput),
				SuggestedAction: "Make sure Docker Desktop is running:\n\n" +
					"1. Open Docker Desktop application\n" +
					"2. Wait for it to start completely\n" +
					"3. Check that the Docker icon in the menu bar shows it's running\n\n" +
					"If issues persist, try restarting Docker Desktop.\n\n" +
					"Docker Desktop guide: https://docs.docker.com/desktop/install/mac-install/",
			}
		}

		// Linux/Unix permission handling
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
				SuggestedAction: fmt.Sprintf("Fix the Docker socket permissions:\n\n"+
					"  sudo chown root:docker /var/run/docker.sock\n"+
					"  sudo chmod 660 /var/run/docker.sock\n\n"+
					"Or restart Docker to recreate the socket:\n\n"+
					"  %s\n\n"+
					"Guide: https://docs.docker.com/engine/install/linux-postinstall/", getDockerRestartCommand()),
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
		SuggestedAction: fmt.Sprintf("Check Docker installation and try:\n\n"+
			"  %s\n\n"+
			"Docker docs: https://docs.docker.com/", getDockerStartCommand()),
	}
}

func checkPodmanService() PreCheckResult {
	cmd := exec.Command("podman", "info")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return PreCheckResult{Passed: true}
	}

	stderrOutput := stderr.String()

	return PreCheckResult{
		Passed:          false,
		ErrorType:       PodmanServiceNotRunning,
		ErrorMessage:    fmt.Sprintf("Podman service is not running or not reachable.\n\nPodman error:\n%s", stderrOutput),
		SuggestedAction: getPodmanErrorMessage(),
	}
}

// Helper function to check if daemon is actually running
func isDaemonRunning() bool {
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}

func RunPreChecks() PreCheckResult {

	// Check - Is runtime configured? If not, prompt user
	if !isRuntimeConfigured() {
		err := promptRuntimeSelection()
		if err != nil {
			return PreCheckResult{
				Passed:          false,
				ErrorType:       NoError,
				ErrorMessage:    fmt.Sprintf("Failed to select runtime: %v", err),
				SuggestedAction: "If you want to Change the runtime, run: \n dockmate --runtime \n",
			}
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return PreCheckResult{
			Passed:          false,
			ErrorType:       NoError,
			ErrorMessage:    fmt.Sprintf("Failed to load config: %v", err),
			SuggestedAction: "Try running:\n  dockmate --runtime\n\nOr delete your config file and try again:\n  rm ~/.config/dockmate/config.yml",
		}
	}

	runtimeType := strings.TrimSpace(strings.ToLower(cfg.Runtime.Type))
	if runtimeType == "" {
		runtimeType = "docker"
	}

	errorChangeRuntimeSuggestion := func(str string) string {
		changeRuntimeSuggestion := "\n\nOr If you want to Change the runtime to " + str + ", run: \n dockmate --runtime \n"
		return changeRuntimeSuggestion
	}

	switch runtimeType {
	case "podman":
		// 1. Check if installed first
		if cfg.Runtime.RunPreChecks {
			result := checkPodmanInstalled()
			if !result.Passed {
				result.SuggestedAction += errorChangeRuntimeSuggestion("docker")
				return result
			}
		}

		// 2. Check Service/Daemon
		result := checkPodmanService()
		if !result.Passed {
			result.SuggestedAction += errorChangeRuntimeSuggestion("docker")
			return result
		}

	case "docker", "auto":
		// 1. Check if installed first
		if cfg.Runtime.RunPreChecks {
			result := checkDockerInstalled()
			if !result.Passed {
				result.SuggestedAction += errorChangeRuntimeSuggestion("podman")
				return result
			}
		}

		// 2. Check Daemon
		result := checkDockerDaemon()
		if !result.Passed {
			result.SuggestedAction += errorChangeRuntimeSuggestion("podman")
			return result
		}

	default:
		return PreCheckResult{
			Passed:          false,
			ErrorType:       NoError,
			ErrorMessage:    fmt.Sprintf("Unsupported runtime type: %s", runtimeType),
			SuggestedAction: "Please choose docker or podman using: \n dockmate --runtime \n",
		}
	}

	// save to config that prechecks have passed (if needed in future)
	cfg.Runtime.RunPreChecks = false
	if err := cfg.Save(); err != nil {
		// log but don't fail prechecks
		fmt.Fprintf(os.Stderr, "Warning: failed to save config after prechecks: %v\n", err)
	}
	return PreCheckResult{Passed: true}
}
