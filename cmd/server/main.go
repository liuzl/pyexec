package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/liuzl/pyexec"
)

func main() {
	flag.Parse()
	// Register the handler from pyexec package
	// It will handle requests like /execute/hello.py
	http.HandleFunc("/execute/", pyexec.HandlePythonExecutionRequestWithUV) // Note the trailing slash

	port := "8080"
	fmt.Printf("Starting server on port %s...\n", port)
	fmt.Printf("Test URL: http://localhost:%s/execute/hello.py?--name=Tester&--verbose\n", port)

	// Start the server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
}
