package pyexec

// Arg represents a command-line argument as a key-value pair.
// This structure is used to preserve the order of arguments.
type Arg struct {
	Key   string
	Value string
}
