package pyexec

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"zliu.org/goutil/rest"
)

var (
	zlog *zerolog.Logger
	once sync.Once
)

func GetZlog() *zerolog.Logger {
	once.Do(func() {
		zlog = rest.Log()
	})
	return zlog
}

func handleExecutionRequest(w http.ResponseWriter, r *http.Request, f func(scriptName string, args []Arg) ([]byte, error)) {
	GetZlog().Info().Str("addr", r.RemoteAddr).Str("method", r.Method).Str("host", r.Host).Str("uri", r.RequestURI).Str("func", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()).Msg("handleExecutionRequest")
	// Extract script name from URL path
	// Example: /execute/my_script.py -> my_script.py
	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[len(pathParts)-1] == "" {
		zlog.Error().Str("url", r.URL.Path).Msg("Script name missing in URL path")
		rest.ErrBadRequest(w, fmt.Sprintf("Script name missing in URL path. Expected format: /%s/<script_name.py>", r.URL.Path))
		return
	}
	scriptName := pathParts[len(pathParts)-1]

	// Extract arguments from query parameters, preserving order
	queryParams := r.URL.Query()
	args := make([]Arg, 0, len(queryParams))
	for key, values := range queryParams {
		// Use the first value for each key. Handles cases like ?flag or ?key=value
		if len(values) > 0 {
			args = append(args, Arg{Key: key, Value: values[0]})
		} else {
			// Handle flags without values (e.g., ?verbose)
			args = append(args, Arg{Key: key, Value: ""})
		}
	}

	// Execute the script
	output, err := f(scriptName, args)
	if err != nil {
		zlog.Error().Str("url", r.URL.Path).Str("error", err.Error()).Msg("Failed to execute script")
		rest.ErrInternalServer(w, fmt.Sprintf("Failed to execute script: %s", err.Error()))
		return
	}

	rest.MustWriteJSONBytes(w, output)
}

// HandlePythonExecutionRequest is an HTTP handler that executes a Python script.
// It expects the script name as the last part of the URL path (e.g., /execute/script.py)
// and arguments as query parameters.
// Example: GET /execute/my_script.py?--input=data.csv&--threshold=0.5
func HandlePythonExecutionRequest(w http.ResponseWriter, r *http.Request) {
	handleExecutionRequest(w, r, ExecutePythonScript)
}

// HandlePythonExecutionRequestWithUV is an HTTP handler that executes a Python script using uv.
// It expects the script name as the last part of the URL path (e.g., /execute/script.py)
// and arguments as query parameters.
// Example: GET /execute/my_script.py?--input=data.csv&--threshold=0.5
func HandlePythonExecutionRequestWithUV(w http.ResponseWriter, r *http.Request) {
	handleExecutionRequest(w, r, ExecutePythonScriptWithUV)
}
