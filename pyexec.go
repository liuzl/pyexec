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
	if err != nil {
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
// Sets the script's working directory to its own directory and runs Python in unbuffered mode.
// Arguments are provided as a map, where keys are flags (e.g., "--model")
// and values are the corresponding flag values. Flags without values can be
// represented with an empty string value.
// It returns the standard output of the script as bytes.
func ExecutePythonScript(scriptName string, args []Arg) ([]byte, error) {
	scriptPath, err := findScript(scriptName)
	if err != nil {
		return nil, fmt.Errorf("failed to find python script: %w", err)
	}

	pythonCmd := getPythonCommand()

	// Prepare command arguments, adding -u for unbuffered output
	cmdArgs := []string{"-u", scriptPath} // <--- Added "-u"
	for _, arg := range args {
		cmdArgs = append(cmdArgs, arg.Key)
		if arg.Value != "" {
			cmdArgs = append(cmdArgs, arg.Value)
		}
	}

	cmd := exec.Command(pythonCmd, cmdArgs...)
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

// ExecutePythonScriptRealtime runs a Python script in unbuffered mode,
// prints its stdout and stderr in real-time, sets the script's working directory
// to its own directory, and returns the complete stdout content.
// It streams the output directly to the Go program's stdout and stderr.
// Returns the captured stdout and an error if the script fails to start or exits with a non-zero status.
func ExecutePythonScriptRealtime(scriptName string, args []Arg) ([]byte, error) {
	scriptPath, err := findScript(scriptName)
	if err != nil {
		return nil, fmt.Errorf("failed to find python script: %w", err)
	}

	pythonCmd := getPythonCommand()

	// Prepare command arguments, adding -u for unbuffered output
	cmdArgs := []string{"-u", scriptPath} // <--- Added "-u"
	for _, arg := range args {
		cmdArgs = append(cmdArgs, arg.Key)
		if arg.Value != "" {
			cmdArgs = append(cmdArgs, arg.Value)
		}
	}

	cmd := exec.Command(pythonCmd, cmdArgs...)
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
