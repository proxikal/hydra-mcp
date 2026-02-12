#!/usr/bin/env python3
"""
MCP server that ignores SIGTERM signals.
Used to test Hydra's SIGKILL fallback mechanism.
"""

import json
import signal
import sys
import time


def ignore_sigterm(signum, frame):
    """Ignore SIGTERM signal - force Hydra to use SIGKILL."""
    sys.stderr.write(json.dumps({
        "level": "info",
        "message": "Ignoring SIGTERM signal",
        "signal": signum
    }) + "\n")
    sys.stderr.flush()
    # Do nothing - ignore the signal


def main():
    # Install SIGTERM handler that ignores the signal
    signal.signal(signal.SIGTERM, ignore_sigterm)

    sys.stderr.write(json.dumps({
        "level": "info",
        "message": "Hanging server started - SIGTERM handler installed"
    }) + "\n")
    sys.stderr.flush()

    # Send initialize response
    for line in sys.stdin:
        try:
            req = json.loads(line.strip())

            if req.get("method") == "initialize":
                response = {
                    "jsonrpc": "2.0",
                    "id": req.get("id"),
                    "result": {
                        "protocolVersion": "2024-11-05",
                        "capabilities": {
                            "tools": {}
                        },
                        "serverInfo": {
                            "name": "hanging-server",
                            "version": "1.0.0"
                        }
                    }
                }
                print(json.dumps(response), flush=True)

                sys.stderr.write(json.dumps({
                    "level": "info",
                    "message": "Initialize response sent"
                }) + "\n")
                sys.stderr.flush()

            elif req.get("method") == "tools/list":
                response = {
                    "jsonrpc": "2.0",
                    "id": req.get("id"),
                    "result": {
                        "tools": []
                    }
                }
                print(json.dumps(response), flush=True)

            elif req.get("method") == "ping":
                response = {
                    "jsonrpc": "2.0",
                    "id": req.get("id"),
                    "result": {"status": "alive"}
                }
                print(json.dumps(response), flush=True)

        except json.JSONDecodeError:
            continue
        except Exception as e:
            sys.stderr.write(json.dumps({
                "level": "error",
                "message": f"Error processing request: {e}"
            }) + "\n")
            sys.stderr.flush()


if __name__ == "__main__":
    main()
