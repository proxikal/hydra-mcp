#!/usr/bin/env python3
"""Chatty server for chaos testing.

Spams stderr with 100 logs/second to test rate limiting.
Tests Hydra's wallet guard protection against token bombs.
"""
import json
import sys
import time
import threading

def spam_logs():
    """Spam stderr with massive log messages."""
    while True:
        # Send 1KB log message
        print("DEBUG: " + "X" * 1000, file=sys.stderr, flush=True)
        time.sleep(0.01)  # 100 logs/second

def main():
    """MCP server that spams logs while handling requests."""
    # Start log spam thread
    spam_thread = threading.Thread(target=spam_logs, daemon=True)
    spam_thread.start()

    # Handle initialize
    line = sys.stdin.readline()
    try:
        req = json.loads(line)
        resp = {
            "jsonrpc": "2.0",
            "id": req["id"],
            "result": {
                "protocolVersion": "1.0",
                "capabilities": {}
            }
        }
        print(json.dumps(resp), flush=True)
    except (json.JSONDecodeError, KeyError):
        pass

    # Echo remaining requests
    for line in sys.stdin:
        try:
            req = json.loads(line)
            resp = {
                "jsonrpc": "2.0",
                "id": req.get("id"),
                "result": req.get("params", {})
            }
            print(json.dumps(resp), flush=True)
        except json.JSONDecodeError:
            pass

if __name__ == "__main__":
    main()
