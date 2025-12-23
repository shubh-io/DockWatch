// internal/config/config.go

package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Layout      LayoutConfig      `yaml:"layout"`
	Performance PerformanceConfig `yaml:"performance"`
	Runtime     RuntimeConfig     `yaml:"runtime"`
	Exec        ExecConfig        `yaml:"exec"`
}

type LayoutConfig struct {
	ContainerId        int `yaml:"container_id_width"`
	ContainerNameWidth int `yaml:"container_name_width"`
	MemoryWidth        int `yaml:"memory_width"`
	CPUWidth           int `yaml:"cpu_width"`
	NetIOWidth         int `yaml:"net_io_width"`
	DiskIOWidth        int `yaml:"disk_io_width"`
	ImageWidth         int `yaml:"image_width"`
	StatusWidth        int `yaml:"status_width"`
	PortWidth          int `yaml:"port_width"`
}

type PerformanceConfig struct {
	PollRate int `yaml:"poll_rate"` // seconds
}

type RuntimeConfig struct {
	Type   string `yaml:"type"`   // "docker" or "podman"
	Socket string `yaml:"socket"` // custom socket path (would add in future)
}

type ExecConfig struct {
	Shell string `yaml:"shell"` // preferred shell for container exec
}

// Default config
func DefaultConfig() *Config {
	return &Config{
		//  8%  CONTAINER ID
		//  14%  NAME
		//   6%  MEMORY
		//   6%  CPU
		//  10%  NET I/O
		//  12%  Disk I/O
		//  18%  IMAGE
		//  13%  STATUS
		//  13%  PORTS
		Layout: LayoutConfig{
			ContainerId:        8,
			ContainerNameWidth: 14,
			MemoryWidth:        6,
			CPUWidth:           6,
			NetIOWidth:         10,
			DiskIOWidth:        12,
			ImageWidth:         18,
			StatusWidth:        13,
			PortWidth:          13,
		},
		Performance: PerformanceConfig{
			PollRate: 2,
		},
		Runtime: RuntimeConfig{
			Type: "docker",
			// optional, would add support later for custom sockets
			Socket: "",
		},
		Exec: ExecConfig{
			Shell: "/bin/sh",
		},
	}
}

// Get config path
func GetConfigPath() (string, error) {
	// Try XDG_CONFIG_HOME first
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "dockmate", "config.yml"), nil
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "dockmate", "config.yml"), nil
}

// Load config
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	// If file doesn't exist, return default
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), nil
	}

	// Parse YAML
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		// If YAML is invalid, return default config
		return DefaultConfig(), nil
	}

	// Apply defaults for missing fields
	if cfg.Exec.Shell == "" {
		cfg.Exec.Shell = "/bin/sh"
	}

	return cfg, nil
}

// Save config
func (c *Config) Save() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// Write file
	return os.WriteFile(path, data, 0644)
}
