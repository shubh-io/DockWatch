package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "docker", cfg.Runtime.Type)
	assert.Equal(t, "", cfg.Runtime.Socket)
	assert.Equal(t, "/bin/sh", cfg.Exec.Shell)
	assert.Equal(t, 2, cfg.Performance.PollRate)
	assert.Equal(t, 8, cfg.Layout.ContainerId)
}

func TestLoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "/bin/sh", cfg.Exec.Shell)
	assert.Equal(t, "docker", cfg.Runtime.Type)
}

func TestLoadWithShell(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "dockmate")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configContent := `
runtime:
  type: podman
  socket: ""
exec:
  shell: /bin/zsh
performance:
  poll_rate: 5
`
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "/bin/zsh", cfg.Exec.Shell)
	assert.Equal(t, "podman", cfg.Runtime.Type)
	assert.Equal(t, 5, cfg.Performance.PollRate)
}

func TestLoadWithoutShell(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "dockmate")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configContent := `
runtime:
  type: docker
  socket: ""
performance:
  poll_rate: 3
`
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "/bin/sh", cfg.Exec.Shell)
	assert.Equal(t, "docker", cfg.Runtime.Type)
	assert.Equal(t, 3, cfg.Performance.PollRate)
}

func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := &Config{
		Layout: LayoutConfig{
			ContainerId:        10,
			ContainerNameWidth: 15,
			MemoryWidth:        7,
			CPUWidth:           7,
			NetIOWidth:         11,
			DiskIOWidth:        13,
			ImageWidth:         19,
			StatusWidth:        14,
			PortWidth:          14,
		},
		Performance: PerformanceConfig{
			PollRate: 4,
		},
		Runtime: RuntimeConfig{
			Type:   "podman",
			Socket: "/custom/socket",
		},
		Exec: ExecConfig{
			Shell: "/bin/bash",
		},
	}

	err := cfg.Save()
	require.NoError(t, err)

	loaded, err := Load()
	require.NoError(t, err)

	assert.Equal(t, cfg.Runtime.Type, loaded.Runtime.Type)
	assert.Equal(t, cfg.Runtime.Socket, loaded.Runtime.Socket)
	assert.Equal(t, cfg.Exec.Shell, loaded.Exec.Shell)
	assert.Equal(t, cfg.Performance.PollRate, loaded.Performance.PollRate)
	assert.Equal(t, cfg.Layout.ContainerId, loaded.Layout.ContainerId)
}

func TestGetConfigPath(t *testing.T) {
	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")

		path, err := GetConfigPath()

		require.NoError(t, err)
		assert.Equal(t, "/custom/config/dockmate/config.yml", path)
	})

	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")

		path, err := GetConfigPath()

		require.NoError(t, err)
		home, _ := os.UserHomeDir()
		assert.Equal(t, filepath.Join(home, ".config", "dockmate", "config.yml"), path)
	})
}

func TestLoadInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "dockmate")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644))

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "/bin/sh", cfg.Exec.Shell)
	assert.Equal(t, "docker", cfg.Runtime.Type)
}
