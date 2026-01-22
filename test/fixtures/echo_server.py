#!/usr/bin/env python3
import json
import sys


def main():
    for line in sys.stdin:
        req = json.loads(line)
        if req.get("method") == "initialize":
            resp = {
                "jsonrpc": "2.0",
                "id": req["id"],
                "result": {
                    "protocolVersion": "1.0",
                    "serverInfo": {"name": "echo"},
                    "capabilities": {"tools": True},
                },
            }
        else:
            resp = {
                "jsonrpc": "2.0",
                "id": req.get("id"),
                "result": req.get("params", {}),
            }
        print(json.dumps(resp), flush=True)


if __name__ == "__main__":
    main()
