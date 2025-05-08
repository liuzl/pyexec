package pyexec

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

func ExecutePythonScriptWithUV(scriptName string, args map[string]string) ([]byte, error) {
	if err := EnsureUVInstalled(); err != nil {
		return nil, fmt.Errorf("failed to ensure uv is installed: %w", err)
	}
	scriptPath, err := findScript(scriptName)
	if err != nil {
		return nil, fmt.Errorf("failed to find python script: %w", err)
	}

	cmdArgs := []string{"run", "--", "python", "-u", scriptPath} // <--- Added "-u"
	for key, value := range args {
		cmdArgs = append(cmdArgs, key)
		if value != "" {
			cmdArgs = append(cmdArgs, value)
		}
	}

	cmd := exec.Command("uv", cmdArgs...)
	cmd.Dir = filepath.Dir(scriptPath)
	GetZlog().Info().Str("cmd", cmd.String()).Msg("Executing command")
	stdout, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}

		errMsg := fmt.Sprintf("python script '%s' (in dir %s) execution failed: %v", scriptName, cmd.Dir, err)
		if stderr != "" {
			errMsg += fmt.Sprintf("\nstderr: %s", stderr)
		}
		if len(stdout) > 0 {
			errMsg += fmt.Sprintf("\nstdout: %s", string(stdout))
		}
		return nil, errors.New(errMsg)
	}

	return stdout, nil
}

func ExecutePythonScriptRealtimeWithUV(scriptName string, args map[string]string) ([]byte, error) {
	if err := EnsureUVInstalled(); err != nil {
		return nil, fmt.Errorf("failed to ensure uv is installed: %w", err)
	}
	scriptPath, err := findScript(scriptName)
	if err != nil {
		return nil, fmt.Errorf("failed to find python script: %w", err)
	}

	cmdArgs := []string{"run", "--", "python", "-u", scriptPath} // <--- Added "-u"
	for key, value := range args {
		cmdArgs = append(cmdArgs, key)
		if value != "" {
			cmdArgs = append(cmdArgs, value)
		}
	}

	cmd := exec.Command("uv", cmdArgs...)
	cmd.Dir = filepath.Dir(scriptPath)

	GetZlog().Info().Str("cmd", cmd.String()).Msg("Executing command")
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	var stdoutBuf bytes.Buffer

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start python script '%s' in dir '%s': %w", scriptName, cmd.Dir, err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		tee := io.TeeReader(stdoutPipe, &stdoutBuf)
		scanner := bufio.NewScanner(tee)
		for scanner.Scan() {
			fmt.Fprintln(os.Stdout, "[stdout]", scanner.Text()) // Now should print in real-time
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error reading stdout from %s: %v\n", scriptName, err)
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, "[stderr]", scanner.Text()) // Stderr is often unbuffered anyway
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error reading stderr from %s: %v\n", scriptName, err)
		}
	}()

	cmdErr := cmd.Wait()
	wg.Wait()

	if cmdErr != nil {
		return stdoutBuf.Bytes(), fmt.Errorf("python script '%s' (in dir %s) exited with error: %w", scriptName, cmd.Dir, cmdErr)
	}

	return stdoutBuf.Bytes(), nil
}
