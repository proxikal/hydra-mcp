#!/usr/bin/env python3
"""
Demo MCP server for real-world validation with Claude Desktop.
Provides simple tools to demonstrate Hydra's resilience.
"""

import json
import sys
import time
from datetime import datetime


def send_response(response):
    """Send JSON-RPC response to stdout."""
    print(json.dumps(response), flush=True)


def log(message, level="info", **extra):
    """Log to stderr (won't corrupt JSON-RPC on stdout)."""
    log_entry = {
        "level": level,
        "time": datetime.now().isoformat(),
        "message": message,
        **extra
    }
    print(json.dumps(log_entry), file=sys.stderr, flush=True)


def handle_initialize(req_id):
    """Handle MCP initialize request."""
    send_response({
        "jsonrpc": "2.0",
        "id": req_id,
        "result": {
            "protocolVersion": "2024-11-05",
            "capabilities": {
                "tools": {}
            },
            "serverInfo": {
                "name": "hydra-demo-server",
                "version": "1.0.0"
            }
        }
    })
    log("Server initialized")


def handle_tools_list(req_id):
    """List available tools."""
    send_response({
        "jsonrpc": "2.0",
        "id": req_id,
        "result": {
            "tools": [
                {
                    "name": "get_current_time",
                    "description": "Get the current time",
                    "inputSchema": {
                        "type": "object",
                        "properties": {},
                        "required": []
                    }
                },
                {
                    "name": "crash_test",
                    "description": "Intentionally crash the server (for testing Hydra resilience)",
                    "inputSchema": {
                        "type": "object",
                        "properties": {},
                        "required": []
                    }
                }
            ]
        }
    })


def handle_tool_call(req_id, params):
    """Execute a tool."""
    tool_name = params.get("name")

    if tool_name == "get_current_time":
        result = {
            "content": [{
                "type": "text",
                "text": f"Current time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
            }]
        }
        send_response({"jsonrpc": "2.0", "id": req_id, "result": result})
        log(f"Tool called: {tool_name}")

    elif tool_name == "crash_test":
        log("Crash test tool called - server will exit", level="warning")
        send_response({
            "jsonrpc": "2.0",
            "id": req_id,
            "result": {
                "content": [{
                    "type": "text",
                    "text": "Server crashing NOW! Hydra should resurrect the session..."
                }]
            }
        })
        time.sleep(0.1)  # Give response time to send
        log("Exiting intentionally", level="error")
        sys.exit(1)  # Crash!

    else:
        send_response({
            "jsonrpc": "2.0",
            "id": req_id,
            "error": {
                "code": -32601,
                "message": f"Unknown tool: {tool_name}"
            }
        })


def main():
    """Main server loop."""
    log("Demo server starting")

    for line in sys.stdin:
        try:
            req = json.loads(line.strip())
            method = req.get("method")
            req_id = req.get("id")

            if method == "initialize":
                handle_initialize(req_id)
            elif method == "tools/list":
                handle_tools_list(req_id)
            elif method == "tools/call":
                handle_tool_call(req_id, req.get("params", {}))
            else:
                # Echo unknown methods
                send_response({
                    "jsonrpc": "2.0",
                    "id": req_id,
                    "result": {"status": "ok", "method": method}
                })

        except json.JSONDecodeError as e:
            log(f"JSON decode error: {e}", level="error")
        except Exception as e:
            log(f"Error: {e}", level="error")


if __name__ == "__main__":
    main()
