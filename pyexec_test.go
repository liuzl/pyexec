package pyexec

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestExecutePythonScript(t *testing.T) {
	scriptName := "test_script.py" // Assumes test_script.py is in the same directory
	args := []Arg{
		{Key: "--arg1", Value: "value1"},
		{Key: "--flag", Value: ""},
		{Key: "-a", Value: "value2"},
	}

	// Expected arguments as they should appear in sys.argv[1:] in the Python script
	// Order now matters.
	expectedArgsOrdered := []string{
		"--arg1", "value1",
		"--flag",
		"-a", "value2",
	}

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
		if len(receivedArgs) != len(expectedArgsOrdered) {
			t.Errorf("Expected %d arguments, but got %d. Expected: %v, Received: %v", len(expectedArgsOrdered), len(receivedArgs), expectedArgsOrdered, receivedArgs)
		}

		// Check if all expected arguments are present and in order
		if !reflect.DeepEqual(receivedArgs, expectedArgsOrdered) {
			t.Errorf("Mismatch in received arguments.\nExpected (ordered): %v\nReceived:           %v", expectedArgsOrdered, receivedArgs)
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
