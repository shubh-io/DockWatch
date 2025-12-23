package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"strconv"
	"strings"

	"github.com/shubh-io/dockmate/pkg/version"
)

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// getShellCommand returns the appropriate shell command
func getShellCommand() (string, bool) {
	if commandExists("bash") {
		return "bash", true
	}
	if commandExists("sh") {
		return "sh", true
	}

	return "", false
}

// Check if dockmate is installed via Homebrew
func isHomebrewInstall() bool {
	if _, err := exec.LookPath("brew"); err == nil {
		if err := exec.Command("brew", "list", "--versions", "shubh-io/tap/dockmate").Run(); err == nil {
			return true
		}
		if err := exec.Command("brew", "list", "--versions", "dockmate").Run(); err == nil {
			return true
		}
		// Fallback: compare executable path to brew prefix
		exe, err := os.Executable()
		if err == nil {
			prefixOut, pErr := exec.Command("brew", "--prefix").Output()
			if pErr == nil {
				prefix := strings.TrimSpace(string(prefixOut))
				exeLower := strings.ToLower(exe)
				// Common brew locations
				if strings.HasPrefix(exeLower, strings.ToLower(prefix)) ||
					strings.Contains(exeLower, "/cellar/dockmate") ||
					strings.Contains(exeLower, "/opt/homebrew") ||
					strings.Contains(exeLower, "/usr/local/cellar") ||
					strings.Contains(exeLower, ".linuxbrew") ||
					strings.Contains(exeLower, "/home/linuxbrew") {
					return true
				}
			}
		}
	}

	// As a last resort, path heuristics without brew available
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exePath := strings.ToLower(exe)
	homebrewHints := []string{
		"/linuxbrew/",
		"/home/linuxbrew/",
		"/homebrew/",
		"/opt/homebrew/",
		"/usr/local/cellar/",
		"cellar/dockmate",
		".linuxbrew",
	}
	for _, h := range homebrewHints {
		if strings.Contains(exePath, strings.ToLower(h)) {
			return true
		}
	}
	return false
}

func getLatestReleaseTag(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	if strings.TrimSpace(release.TagName) == "" {
		return "", fmt.Errorf("no tag name found in release")
	}

	return release.TagName, nil
}

// trims whitespace and leading 'v' or 'V'
func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if strings.HasPrefix(tag, "v") || strings.HasPrefix(tag, "V") {
		return tag[1:]
	}
	return tag
}

func compareSemver(a, b string) int {
	a = normalizeTag(a)
	b = normalizeTag(b)
	if a == b {
		return 0
	}
	// split a into parts - eg: "1.2.3" -> ["1","2","3"]
	a_splited := strings.Split(a, ".")
	// split b into parts - eg: "1.2.0" -> ["1","2","0"]
	b_splited := strings.Split(b, ".")

	// compare each part
	n := len(a_splited)

	if len(b_splited) > n {
		n = len(b_splited)
	}

	for i := 0; i < n; i++ {
		var a_value, b_value string
		if i < len(a_splited) {
			a_value = a_splited[i]
		}
		if i < len(b_splited) {
			b_value = b_splited[i]
		}
		if a_value == b_value {
			continue
		}
		// attempting numeric compare for best accuracy
		ai, aErr := strconv.Atoi(a_value)
		bi, bErr := strconv.Atoi(b_value)
		if aErr == nil && bErr == nil {
			if ai < bi {
				return -1
			}
			if ai > bi {
				return 1
			}

			continue
		}

		if cmp := strings.Compare(a_value, b_value); cmp != 0 {
			return cmp
		}
	}
	return 0
}

func UpdateCommand() {
	fmt.Println("Checking for updates...")

	// Check if installed via Homebrew FIRST
	if isHomebrewInstall() {
		fmt.Println("⚠️ Detected: dockmate is installed via Homebrew")
		fmt.Println("")
		fmt.Println("To update, please run:")
		fmt.Println("  brew upgrade shubh-io/tap/dockmate")
		fmt.Println("")
		fmt.Println("Current version:", version.Dockmate_Version)
		return
	}

	current := version.Dockmate_Version

	latestTag, err := getLatestReleaseTag(version.Repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not check latest release: %v\n", err)
		return
	}

	// compare normalized tags (striped 'v')
	cmp := compareSemver(current, latestTag)
	if cmp >= 0 {
		fmt.Printf("Already up-to-date (current: %s, latest: %s)\n", current, latestTag)
		return
	}

	fmt.Printf("New release available! : %s → %s\n", current, latestTag)
	fmt.Println("Re-running installer to update...")

	// Check for required shell
	_, hasShell := getShellCommand()
	if !hasShell {
		fmt.Fprintln(os.Stderr, "Error: No compatible shell found (bash, sh)")
		fmt.Fprintln(os.Stderr, "Please install bash or sh to use auto-update")
		fmt.Printf("\nManual update: https://github.com/%s/releases/latest\n", version.Repo)
		return
	}

	installURL := "https://raw.githubusercontent.com/shubh-io/dockmate/main/install.sh"
	installScript := "install.sh"

	// Try piped install first using `sh` only for portability.

	if commandExists("sh") {
		if commandExists("curl") {
			cmd := exec.Command("sh", "-c", fmt.Sprintf("curl -fsSL %s | sh", installURL))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				fmt.Println("")
				fmt.Println("Updated successfully!")
				return
			}
			fmt.Println("Piped install failed, trying fallback method...")
		} else if commandExists("wget") {
			cmd := exec.Command("sh", "-c", fmt.Sprintf("wget -qO- %s | sh", installURL))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				fmt.Println("")
				fmt.Println("Updated successfully!")
				return
			}
			fmt.Println("Piped install failed, trying fallback method...")
		}
	}

	fmt.Println("Downloading installer...")
	if err := downloadFile(installURL, installScript); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to download install script: %v\n", err)
		fmt.Printf("\nPlease update manually: https://github.com/%s/releases/latest\n", version.Repo)
		return
	}
	// run installer script
	fmt.Println("Running installer...")
	runCmd := exec.Command("sh", installScript)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	if err := runCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		fmt.Printf("\nPlease update manually: https://github.com/%s/releases/latest\n", version.Repo)
		// Still try to clean up
		os.Remove(installScript)
		return
	}

	// removes the script file
	if err := os.Remove(installScript); err != nil {
		fmt.Printf("Warning: could not remove %s: %v\n", installScript, err)
	}

	fmt.Println("")
	fmt.Println("Updated successfully!")
}
