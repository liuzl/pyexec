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
	"runtime"
	"strings"
	"sync"
)

// fileExists checks if a file exists and is not a directory.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// findScript attempts to locate the specified Python script by its name.
// It searches in environment variables, a configurable list of directories,
// relative paths, near the executable, and near the caller's source file.
func findScript(scriptName string) (string, error) {
	// 1. Check specific environment variable (e.g., SCRIPTNAME_PATH)
	envVarSpecific := strings.ToUpper(strings.ReplaceAll(scriptName, ".", "_")) + "_PATH"
	if scriptPath := os.Getenv(envVarSpecific); scriptPath != "" && fileExists(scriptPath) {
		absPath, err := filepath.Abs(scriptPath)
		if err == nil {
			return absPath, nil
		}
		return scriptPath, nil // Return original path if abs fails
	}

	// 2. Check directories specified in PYEXEC_SCRIPT_DIRS environment variable
	envVarDirs := "PYEXEC_SCRIPT_DIRS"
	if searchDirs := os.Getenv(envVarDirs); searchDirs != "" {
		dirList := filepath.SplitList(searchDirs) // Handles OS-specific separator ( : or ; )
		for _, dir := range dirList {
			scriptPath := filepath.Join(dir, scriptName)
			if fileExists(scriptPath) {
				absPath, err := filepath.Abs(scriptPath)
				if err == nil {
					return absPath, nil
				}
				return scriptPath, nil // Return found path if abs fails
			}
		}
	}

	// 3. Try paths relative to the current working directory
	possiblePaths := []string{
		scriptName,
		filepath.Join(".", scriptName),
		// Add other common relative structures if needed
		// e.g., filepath.Join(".", "scripts", scriptName),
	}

	// 4. Try paths relative to the executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths = append(possiblePaths,
			filepath.Join(execDir, scriptName),
			// e.g., filepath.Join(execDir, "scripts", scriptName),
		)
	}

	// 5. Try paths relative to the caller's source file (useful for tests)
	_, currentFile, _, ok := runtime.Caller(1) // Caller(1) to get the caller of findScript
	if ok {
		currentDir := filepath.Dir(currentFile)
		possiblePaths = append(possiblePaths,
			filepath.Join(currentDir, scriptName),
			filepath.Join(currentDir, "..", scriptName),
			// e.g., filepath.Join(currentDir, "..", "scripts", scriptName),
		)
	}

	// Check all calculated possible paths (from steps 3, 4, 5)
	for _, path := range possiblePaths {
		cleanPath := filepath.Clean(path)
		if fileExists(cleanPath) {
			absPath, err := filepath.Abs(cleanPath)
			if err == nil {
				return absPath, nil
			}
			return cleanPath, nil // Return relative path if abs fails
		}
	}

	return "", fmt.Errorf("script '%s' not found in any of the expected locations (checked env %s, env %s, cwd, executable dir, caller dir)", scriptName, envVarSpecific, envVarDirs)
}

// getPythonCommand returns the appropriate Python command (python3 or python).
// It checks the PYTHON_COMMAND environment variable first.
func getPythonCommand() string {
	// Check environment variable
	if pythonCmd := os.Getenv("PYTHON_COMMAND"); pythonCmd != "" {
		if _, err := exec.LookPath(pythonCmd); err == nil {
			return pythonCmd
		}
		fmt.Printf("Warning: PYTHON_COMMAND environment variable set to '%s', but command not found. Trying defaults.\n", pythonCmd)
	}

	// Try python3 first
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}

	// Fallback to python
	if _, err := exec.LookPath("python"); err == nil {
		return "python"
	}

	// If neither is found, return a default and let exec.Command fail later
	return "python"
}

// ExecutePythonScript runs a specified Python script with given arguments.
// Arguments are provided as a map, where keys are flags (e.g., "--model")
// and values are the corresponding flag values. Flags without values can be
// represented with an empty string value.
// It returns the standard output of the script as bytes.
func ExecutePythonScript(scriptName string, args map[string]string) ([]byte, error) {
	// Find the script path
	scriptPath, err := findScript(scriptName)
	if err != nil {
		return nil, fmt.Errorf("failed to find python script: %w", err)
	}

	// Get the Python command
	pythonCmd := getPythonCommand()

	// Prepare command arguments
	cmdArgs := []string{scriptPath}
	for key, value := range args {
		cmdArgs = append(cmdArgs, key)
		// Only append value if it's not empty (allows for boolean flags like --verbose)
		if value != "" {
			cmdArgs = append(cmdArgs, value)
		}
	}

	// Create the command
	cmd := exec.Command(pythonCmd, cmdArgs...)

	// Execute the command and capture stdout and stderr
	stdout, err := cmd.Output() // cmd.Output() captures stdout and returns stderr in the error
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}

		errMsg := fmt.Sprintf("python script '%s' execution failed: %v", scriptName, err)
		if stderr != "" {
			errMsg += fmt.Sprintf("stderr: %s", stderr)
		}
		// Include stdout in error message if available, as it might contain partial output or script-level errors
		if len(stdout) > 0 {
			errMsg += fmt.Sprintf("stdout: %s", string(stdout))
		}
		return nil, errors.New(errMsg)
	}

	return stdout, nil
}

// ExecutePythonScriptRealtime runs a Python script, prints its stdout and stderr in real-time,
// and returns the complete stdout content.
// It streams the output directly to the Go program's stdout and stderr.
// Returns the captured stdout and an error if the script fails to start or exits with a non-zero status.
func ExecutePythonScriptRealtime(scriptName string, args map[string]string) ([]byte, error) {
	scriptPath, err := findScript(scriptName)
	if err != nil {
		return nil, fmt.Errorf("failed to find python script: %w", err)
	}

	pythonCmd := getPythonCommand()

	cmdArgs := []string{scriptPath}
	for key, value := range args {
		cmdArgs = append(cmdArgs, key)
		if value != "" {
			cmdArgs = append(cmdArgs, value)
		}
	}

	cmd := exec.Command(pythonCmd, cmdArgs...)

	// Get pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Buffer to capture stdout
	var stdoutBuf bytes.Buffer

	// Start the command asynchronously
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start python script '%s': %w", scriptName, err)
	}

	// Use a WaitGroup to wait for pipe reading goroutines to finish
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine to read, print, and capture stdout
	go func() {
		defer wg.Done()
		// Combine writing to buffer and printing to os.Stdout
		tee := io.TeeReader(stdoutPipe, &stdoutBuf) // Reads from stdoutPipe, writes to stdoutBuf
		scanner := bufio.NewScanner(tee)
		for scanner.Scan() {
			fmt.Fprintln(os.Stdout, "[stdout]", scanner.Text()) // Print in real-time
			// Data is already written to stdoutBuf by io.TeeReader implicitly
		}
		if err := scanner.Err(); err != nil {
			// Still print errors related to reading stdout
			fmt.Fprintf(os.Stderr, "error reading stdout from %s: %v\n", scriptName, err)
		}
	}()

	// Goroutine to read and print stderr (no capture needed)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, "[stderr]", scanner.Text()) // Prefix for clarity
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error reading stderr from %s: %v\n", scriptName, err)
		}
	}()

	// Wait for the command to finish
	cmdErr := cmd.Wait() // Capture the potential error from the command exit status

	// Wait for the reading goroutines to finish *after* the command has exited
	wg.Wait()

	if cmdErr != nil {
		// Return captured stdout along with the error
		return stdoutBuf.Bytes(), fmt.Errorf("python script '%s' exited with error: %w", scriptName, cmdErr)
	}

	// Return captured stdout and nil error on success
	return stdoutBuf.Bytes(), nil
}
