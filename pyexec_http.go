package pyexec

import (
	"fmt"
	"net/http"
	"net/url"
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

	// Extract arguments from raw query parameters to preserve order
	var args []Arg
	if r.URL.RawQuery != "" {
		rawParams := strings.Split(r.URL.RawQuery, "&")
		args = make([]Arg, 0, len(rawParams))
		for _, param := range rawParams {
			if param == "" { // Skip empty parameters (e.g., from "&&" or trailing "&")
				continue
			}
			var key, value string
			parts := strings.SplitN(param, "=", 2)

			decodedKey, err := url.QueryUnescape(parts[0])
			if err != nil {
				GetZlog().Warn().Str("raw_key", parts[0]).Err(err).Msg("Failed to unescape query parameter key")
				rest.ErrBadRequest(w, fmt.Sprintf("Malformed query parameter key: %s", parts[0]))
				return
			}
			key = decodedKey

			if len(parts) == 2 {
				decodedValue, err := url.QueryUnescape(parts[1])
				if err != nil {
					GetZlog().Warn().Str("raw_value", parts[1]).Err(err).Msg("Failed to unescape query parameter value")
					rest.ErrBadRequest(w, fmt.Sprintf("Malformed query parameter value for key %s: %s", key, parts[1]))
					return
				}
				value = decodedValue
			} else {
				value = "" // No value part, so it's a flag
			}
			args = append(args, Arg{Key: key, Value: value})
		}
	}
	if args == nil { // Ensure args is an empty slice if RawQuery was empty
		args = make([]Arg, 0)
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
