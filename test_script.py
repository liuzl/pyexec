import sys
import json

if __name__ == "__main__":
    # Skip the script name itself (sys.argv[0])
    args_to_print = sys.argv[1:]
    print(json.dumps(args_to_print)) 