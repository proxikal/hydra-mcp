#!/usr/bin/env python3
"""Slow startup server for chaos testing.

Simulates a server that takes 30 seconds to initialize.
Tests Hydra's timeout handling and queued request replay.
"""
import json
import sys
import time

# Simulate slow startup (30 seconds)
time.sleep(30)

def main():
    """Echo server that responds after slow startup."""
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
            # Ignore invalid JSON
            pass

if __name__ == "__main__":
    main()
