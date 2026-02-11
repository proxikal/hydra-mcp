#!/usr/bin/env python3
"""
Chaos MCP Server (Python)
Intentionally unstable server for testing Hydra's fault tolerance.
"""
import sys
import json
import time
import random
import signal

# Track request count for deterministic chaos
request_count = 0

def handle_request(msg):
    """Handle a single JSON-RPC request with chaos injection."""
    global request_count
    request_count += 1

    # Chaos behaviors (increasing probability)
    chaos_roll = random.random()

    # 1% chance: Immediate crash
    if chaos_roll < 0.01:
        sys.stderr.write(f"[CHAOS] Request {request_count}: CRASH\n")
        sys.stderr.flush()
        sys.exit(1)

    # 3% chance: Hang for 30 seconds
    if chaos_roll < 0.04:
        sys.stderr.write(f"[CHAOS] Request {request_count}: HANG (30s)\n")
        sys.stderr.flush()
        time.sleep(30)

    # 5% chance: Slow response (2-5 seconds)
    if chaos_roll < 0.09:
        delay = random.uniform(2.0, 5.0)
        sys.stderr.write(f"[CHAOS] Request {request_count}: SLOW ({delay:.1f}s)\n")
        sys.stderr.flush()
        time.sleep(delay)

    # Normal response
    response = {
        "jsonrpc": "2.0",
        "id": msg.get("id"),
        "result": {
            "status": "ok",
            "request_count": request_count,
            "server": "python-chaos"
        }
    }

    print(json.dumps(response), flush=True)

def main():
    """Main server loop with slow initialization."""
    # Simulate slow initialization (2 seconds)
    sys.stderr.write("[CHAOS] Initializing (2s)...\n")
    sys.stderr.flush()
    time.sleep(2)
    sys.stderr.write("[CHAOS] Ready\n")
    sys.stderr.flush()

    # Process requests
    try:
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue

            try:
                msg = json.loads(line)
                handle_request(msg)
            except json.JSONDecodeError as e:
                sys.stderr.write(f"[ERROR] Invalid JSON: {e}\n")
                sys.stderr.flush()
    except KeyboardInterrupt:
        sys.stderr.write("[CHAOS] Shutting down\n")
        sys.stderr.flush()

if __name__ == "__main__":
    main()
