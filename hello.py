import sys
import argparse
import json

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='A simple test script.')
    parser.add_argument('--name', default='World', help='Name to greet')
    parser.add_argument('--verbose', action='store_true', help='Enable verbose output')

    # Parse known args, ignore unknown ones if necessary, or handle them
    # For simplicity, we'll parse known args. The Go handler passes all query params.
    known_args, _ = parser.parse_known_args()

    # Construct the output data as a dictionary
    output_data = {
        "message": f"Hello, {known_args.name}!",
        "verbose": known_args.verbose,
        "arguments_received": sys.argv
    }

    # Print the dictionary as a JSON string
    print(json.dumps(output_data, indent=4)) # Using indent for readability
