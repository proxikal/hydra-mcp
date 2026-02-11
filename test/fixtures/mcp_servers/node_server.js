#!/usr/bin/env node
/**
 * Chaos MCP Server (Node.js)
 * Intentionally unstable server for testing Hydra's fault tolerance.
 */
const readline = require('readline');

let requestCount = 0;

function handleRequest(line) {
  requestCount++;

  try {
    const msg = JSON.parse(line);

    // Chaos behaviors
    const chaosRoll = Math.random();

    // 1% chance: Immediate crash
    if (chaosRoll < 0.01) {
      console.error(`[CHAOS] Request ${requestCount}: CRASH`);
      process.exit(1);
    }

    // 3% chance: Hang for 30 seconds
    if (chaosRoll < 0.04) {
      console.error(`[CHAOS] Request ${requestCount}: HANG (30s)`);
      setTimeout(() => {
        // Eventually respond
        sendResponse(msg);
      }, 30000);
      return;
    }

    // 5% chance: Slow response (2-5 seconds)
    if (chaosRoll < 0.09) {
      const delay = Math.random() * 3000 + 2000;
      console.error(`[CHAOS] Request ${requestCount}: SLOW (${(delay/1000).toFixed(1)}s)`);
      setTimeout(() => {
        sendResponse(msg);
      }, delay);
      return;
    }

    // Normal response
    sendResponse(msg);

  } catch (e) {
    console.error(`[ERROR] Invalid JSON: ${e.message}`);
  }
}

function sendResponse(msg) {
  const response = {
    jsonrpc: "2.0",
    id: msg.id,
    result: {
      status: "ok",
      request_count: requestCount,
      server: "node-chaos"
    }
  };
  console.log(JSON.stringify(response));
}

function main() {
  // Simulate slow initialization (2 seconds)
  console.error('[CHAOS] Initializing (2s)...');

  setTimeout(() => {
    console.error('[CHAOS] Ready');

    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
      terminal: false
    });

    rl.on('line', (line) => {
      if (line.trim()) {
        handleRequest(line);
      }
    });

    rl.on('close', () => {
      console.error('[CHAOS] Shutting down');
      process.exit(0);
    });

  }, 2000);
}

// Handle signals
process.on('SIGTERM', () => {
  console.error('[CHAOS] Received SIGTERM');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.error('[CHAOS] Received SIGINT');
  process.exit(0);
});

main();
