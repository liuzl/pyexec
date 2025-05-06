package pyexec

import (
	"fmt"
	"net/http"
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

// HandlePythonExecutionRequest is an HTTP handler that executes a Python script.
// It expects the script name as the last part of the URL path (e.g., /execute/script.py)
// and arguments as query parameters.
// Example: GET /execute/my_script.py?--input=data.csv&--threshold=0.5
func HandlePythonExecutionRequest(w http.ResponseWriter, r *http.Request) {
	GetZlog().Info().Str("addr", r.RemoteAddr).Str("method", r.Method).Str("host", r.Host).Str("uri", r.RequestURI).Msg("HandlePythonExecutionRequest")
	// Extract script name from URL path
	// Example: /execute/my_script.py -> my_script.py
	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[len(pathParts)-1] == "" {
		zlog.Error().Str("url", r.URL.Path).Msg("Script name missing in URL path")
		rest.ErrBadRequest(w, fmt.Sprintf("Script name missing in URL path. Expected format: /%s/<script_name.py>", r.URL.Path))
		return
	}
	scriptName := pathParts[len(pathParts)-1]

	// Extract arguments from query parameters
	args := make(map[string]string)
	for key, values := range r.URL.Query() {
		// Use the first value for each key. Handles cases like ?flag or ?key=value
		if len(values) > 0 {
			args[key] = values[0]
		} else {
			// Handle flags without values (e.g., ?verbose)
			args[key] = ""
		}
	}

	// Execute the script
	output, err := ExecutePythonScript(scriptName, args)
	if err != nil {
		zlog.Error().Str("url", r.URL.Path).Str("error", err.Error()).Msg("Failed to execute script")
		rest.ErrInternalServer(w, fmt.Sprintf("Failed to execute script: %s", err.Error()))
		return
	}

	rest.MustWriteJSONBytes(w, output)
}
