package pyexec

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// EnsureUVInstalled checks if uv is installed and installs it if not.
func EnsureUVInstalled() error {
	_, err := exec.LookPath("uv")
	if err == nil {
		// uv is already installed
		if zlog != nil {
			zlog.Info().Msg("uv is already installed.")
		}
		return nil
	}

	if zlog != nil {
		zlog.Info().Msg("uv not found, attempting to install.")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux", "darwin":
		// For Linux and macOS
		cmd = exec.Command("sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh")
	case "windows":
		// For Windows
		cmd = exec.Command("powershell", "-Command", "irm https://astral.sh/uv/install.ps1 | iex")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install uv: %w", err)
	}

	if zlog != nil {
		zlog.Info().Msg("uv installed successfully.")
	}
	return nil
}
