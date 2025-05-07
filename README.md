# pyexec

`pyexec` is a Go module designed to execute Python scripts. It offers flexibility in how scripts are discovered and run, including direct execution, execution via the `uv` Python package manager, and an HTTP server to expose script execution as a service.

## Features

*   **Execute Local Python Scripts**: Run Python scripts from your Go applications.
*   **Flexible Script Discovery**: Locates scripts using:
    *   Environment variables (e.g., `SCRIPTNAME_PATH`, `PYEXEC_SCRIPT_DIRS`).
    *   Paths relative to the current working directory.
    *   Paths relative to the Go executable.
    *   Paths relative to the caller's source file (useful for tests).
*   **Python Interpreter Management**:
    *   Automatically attempts to use `python3`.
    *   Falls back to `python` if `python3` is not found.
    *   Configurable via the `PYTHON_COMMAND` environment variable.
*   **Argument Passing**: Pass command-line arguments to Python scripts.
*   **Real-time Output**: Stream `stdout` and `stderr` from Python scripts in real-time.
*   **`uv` Integration**: Execute scripts using `uv run`, facilitating Python environment and dependency management.
*   **HTTP Server**: Expose Python script execution via a REST API.

## Requirements

*   Go `1.24.2` or later.
*   Python 3 (recommended). The system will fall back to `python` if `python3` is not available.
*   `uv` (optional): Required if you intend to use the `uv`-based script execution features.

## Installation

### As a Library

To use `pyexec` as a library in your Go project:
```bash
go get github.com/liuzl/pyexec
```

### HTTP Server Executable

To build the standalone HTTP server:
```bash
go build -o pyexec_server ./cmd/server/main.go
```

## Usage

### As a Go Library

You can call Python scripts directly from your Go code.

```go
package main

import (
	"fmt"
	"log"

	"github.com/liuzl/pyexec"
)

func main() {
	// Example: Execute a script directly
	args := map[string]string{
		"--name": "GoApp",
		"--verbose": "", // For flags without values
	}
	output, err := pyexec.ExecutePythonScript("hello.py", args)
	if err != nil {
		log.Fatalf("Error executing hello.py: %v", err)
	}
	fmt.Printf("Output from hello.py:\n%s\n", string(output))

	// Example: Execute a script with real-time output
	// outputRealtime, err := pyexec.ExecutePythonScriptRealtime("hello.py", args)
	// if err != nil {
	// log.Fatalf("Error executing hello.py with real-time output: %v", err)
	// }
	// fmt.Printf("Captured stdout from hello.py (real-time):\n%s\n", string(outputRealtime))


	// Example: Execute a script using uv (ensure uv is installed and in PATH)
	// outputUV, err := pyexec.ExecutePythonScriptWithUV("hello.py", args)
	// if err != nil {
	// log.Fatalf("Error executing hello.py with uv: %v", err)
	// }
	// fmt.Printf("Output from hello.py (via uv):\n%s\n", string(outputUV))
}
```
*(Note: Ensure `hello.py` is discoverable by `pyexec` as per the "Script Discovery" rules.)*

### HTTP Server

The project includes an HTTP server to execute scripts remotely.

**1. Running the Server:**

You can run the server using `go run` or by building the binary first:
```bash
go run ./cmd/server/main.go
```
Or, if you built the binary (`pyexec_server`):
```bash
./pyexec_server
```
The server will start on port `8080` by default.

**2. Executing Scripts via API:**

*   **Endpoint**: `GET /execute/<script_name.py>`
*   **Script Arguments**: Pass script arguments as URL query parameters.
    *   For arguments with values: `?--argname=value`
    *   For flags (arguments without values): `?--flagname`

**Example using `curl` with `hello.py`:**
```bash
curl "http://localhost:8080/execute/hello.py?--name=Universe&--verbose"
```

This will execute `hello.py`, passing `--name Universe` and `--verbose` as arguments. The script's standard output (expected to be JSON) will be returned in the HTTP response.

### Script Discovery

`pyexec` locates Python scripts in the following order:
1.  Environment variable `<SCRIPT_NAME_AS_VAR>_PATH` (e.g., for `my_script.py`, check `MY_SCRIPT_PY_PATH`).
2.  Directories listed in the `PYEXEC_SCRIPT_DIRS` environment variable (colon-separated on Linux/macOS, semicolon-separated on Windows).
3.  Paths relative to the current working directory (e.g., `script.py`, `./scripts/script.py`).
4.  Paths relative to the Go program's executable.
5.  Paths relative to the caller's source file (mainly for tests).

### Python Command Configuration

By default, `pyexec` tries `python3` first, then `python`. You can specify a particular Python command by setting the `PYTHON_COMMAND` environment variable:
```bash
export PYTHON_COMMAND=/usr/local/bin/python3.9
```

## Example Python Script (`hello.py`)

The `hello.py` script included in this repository is a simple argument parser that outputs JSON:
```python
import sys
import argparse
import json

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='A simple test script.')
    parser.add_argument('--name', default='World', help='Name to greet')
    parser.add_argument('--verbose', action='store_true', help='Enable verbose output')

    known_args, _ = parser.parse_known_args()

    output_data = {
        "message": f"Hello, {known_args.name}!",
        "verbose": known_args.verbose,
        "arguments_received": sys.argv
    }
    print(json.dumps(output_data, indent=4))
```

When called via the HTTP server example (`curl "http://localhost:8080/execute/hello.py?--name=Universe&--verbose"`), it would output:
```json
{
    "message": "Hello, Universe!",
    "verbose": true,
    "arguments_received": [
        "hello.py",
        "--name",
        "Universe",
        "--verbose"
    ]
}
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.
