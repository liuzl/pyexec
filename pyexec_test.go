package pyexec

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestExecutePythonScript(t *testing.T) {
	scriptName := "test_script.py" // Assumes test_script.py is in the same directory
	args := map[string]string{
		"--arg1": "value1",
		"--flag": "", // Test flag without value
		"-a":     "value2",
	}

	// Expected arguments as they should appear in sys.argv[1:] in the Python script
	// Note: Go map iteration order is not guaranteed, so we can't rely on exact order.
	// Instead, we'll check if the output contains all expected arguments.
	expectedArgsSet := map[string]bool{
		"--arg1": true,
		"value1": true,
		"--flag": true,
		"-a":     true,
		"value2": true,
	}
	expectedArgsCount := 5

	t.Run("NormalExecution", func(t *testing.T) {
		stdout, err := ExecutePythonScript(scriptName, args)
		if err != nil {
			t.Fatalf("ExecutePythonScript failed: %v", err)
		}

		// Trim whitespace and parse the JSON output
		outputStr := strings.TrimSpace(string(stdout))
		t.Log(outputStr)
		var receivedArgs []string
		if err := json.Unmarshal([]byte(outputStr), &receivedArgs); err != nil {
			t.Fatalf("Failed to parse JSON output from script: %v\nOutput: %s", err, outputStr)
		}

		// Check if the number of arguments matches
		if len(receivedArgs) != expectedArgsCount {
			t.Errorf("Expected %d arguments, but got %d. Received: %v", expectedArgsCount, len(receivedArgs), receivedArgs)
		}

		// Check if all expected arguments are present
		receivedArgsSet := make(map[string]bool)
		for _, arg := range receivedArgs {
			receivedArgsSet[arg] = true
		}

		if !reflect.DeepEqual(receivedArgsSet, expectedArgsSet) {
			t.Errorf("Mismatch in received arguments.\nExpected presence: %v\nReceived: %v", expectedArgsSet, receivedArgsSet)
		}
	})

	t.Run("ScriptNotFound", func(t *testing.T) {
		_, err := ExecutePythonScript("non_existent_script.py", nil)
		if err == nil {
			t.Fatal("Expected an error when script is not found, but got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected error message to contain 'not found', but got: %v", err)
		}
	})

	// Add more test cases if needed, e.g., script execution error
}
