#!/bin/bash

# ==============================================================================
# Memory Monitoring Script for PMA Backend
# ==============================================================================
# This script starts the Go backend server, redirects its output to a log file,
# and monitors its memory usage. If the memory exceeds a defined threshold,
# it kills the server process to prevent system instability from a memory leak.
#
# Usage:
# ./pma-backend-go/scripts/monitor_memory.sh
#
# The script must be run from the project root directory (PMAutomation/).
# ==============================================================================

# --- Configuration ---
# Path to the Go backend directory, relative to the script execution path
BACKEND_DIR="pma-backend-go"

# Log file for the server output
LOG_FILE="server.log"

# Maximum memory threshold in Kilobytes (KB). 1 GB = 1024 * 1024 KB = 1048576 KB
MAX_MEM_KB=1048576

# Interval between memory checks, in seconds
CHECK_INTERVAL=10

# Maximum runtime in seconds (30 minutes)
MAX_RUNTIME=1800

# --- State ---
SERVER_PID=0
START_TIME=$(date +%s)
INITIAL_MEM_KB=0

# --- Cleanup Function ---
# This function is called when the script exits, ensuring the server is stopped.
cleanup() {
  echo ""
  echo "---"
  if [ -n "$SERVER_PID" ] && ps -p "$SERVER_PID" > /dev/null; then
    echo "🛑 Stopping server process (PID: $SERVER_PID)..."
    # Kill the entire process group to ensure all child processes are stopped
    pkill -P "$SERVER_PID" 2>/dev/null
    kill "$SERVER_PID"
    sleep 2
    if ps -p "$SERVER_PID" > /dev/null; then
      echo "⚠️ Server did not stop gracefully, forcing shutdown with SIGKILL..."
      pkill -9 -P "$SERVER_PID" 2>/dev/null
      kill -9 "$SERVER_PID"
    fi
    echo "✅ Server stopped."
  else
    echo "ℹ️ Server process not running or already stopped."
  fi
  echo "---"
  echo "👋 Exiting monitoring script."
}

# Trap script exit (Ctrl+C, etc.) to run the cleanup function
trap cleanup EXIT

# Function to get total memory usage of a process and all its children
get_total_memory() {
  local pid=$1
  local total_mem=0
  
  # Get memory of the main process
  if [ -f "/proc/$pid/status" ]; then
    local mem=$(grep VmRSS "/proc/$pid/status" | awk '{print $2}')
    if [ -n "$mem" ]; then
      total_mem=$((total_mem + mem))
    fi
  fi
  
  # Get memory of all child processes
  local children=$(pgrep -P "$pid" 2>/dev/null)
  for child in $children; do
    if [ -f "/proc/$child/status" ]; then
      local child_mem=$(grep VmRSS "/proc/$child/status" | awk '{print $2}')
      if [ -n "$child_mem" ]; then
        total_mem=$((total_mem + child_mem))
      fi
    fi
  done
  
  echo "$total_mem"
}

# Function to list all child processes
list_child_processes() {
  local pid=$1
  echo "📋 Child processes of PID $pid:"
  ps -o pid,ppid,cmd --forest -p "$pid" 2>/dev/null || echo "No child processes found"
}

# --- Main Execution ---
# Navigate to the backend directory
cd "$BACKEND_DIR" || { echo "❌ Error: Could not navigate to $BACKEND_DIR. Please run from the project root."; exit 1; }

# Clear previous log file
> "$LOG_FILE"
echo "📝 Cleared previous log file: $LOG_FILE"

echo "🚀 Starting Go backend server in the background..."
# Start the server, redirecting stdout and stderr to the log file.
# Run it in the background (&) and get its Process ID ($!).
go run ./cmd/server >> "$LOG_FILE" 2>&1 &
SERVER_PID=$!

# Check if the server started successfully
sleep 2 # Give it a moment to start or fail
if ! ps -p "$SERVER_PID" > /dev/null; then
  echo "❌ FATAL: Server process failed to start. Check '$LOG_FILE' for errors."
  exit 1
fi

echo "✅ Server started successfully with PID: $SERVER_PID"
echo "---"
echo "🔎 Monitoring memory usage every $CHECK_INTERVAL seconds. Threshold: ${MAX_MEM_KB} KB (~1 GB)."
echo "📊 Monitoring main process AND all child processes"
echo "⏱️  Maximum runtime: $MAX_RUNTIME seconds (30 minutes)"
echo "   (Press Ctrl+C to stop the server and the monitor)"
echo ""

# Get initial memory usage (including children)
INITIAL_MEM_KB=$(get_total_memory "$SERVER_PID")
echo "📊 Initial memory usage (main + children): ${INITIAL_MEM_KB} KB"
list_child_processes "$SERVER_PID"
echo ""

# --- Monitoring Loop ---
while true; do
  # Check if the server process is still running
  if ! ps -p "$SERVER_PID" > /dev/null; then
    echo "ℹ️ Server process (PID: $SERVER_PID) has stopped unexpectedly. Check '$LOG_FILE'."
    break
  fi

  # Check if we've exceeded the maximum runtime
  CURRENT_TIME=$(date +%s)
  ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
  if [ $ELAPSED_TIME -gt $MAX_RUNTIME ]; then
    echo "⏰ Maximum runtime reached ($MAX_RUNTIME seconds). Stopping monitoring."
    break
  fi

  # Get current memory usage (including all child processes)
  CURRENT_MEM_KB=$(get_total_memory "$SERVER_PID")

  if [ -z "$CURRENT_MEM_KB" ] || [ "$CURRENT_MEM_KB" -eq 0 ]; then
    # This can happen if the process terminates between the `ps` check and `grep`
    echo "Could not retrieve memory usage. Process might have just terminated."
    sleep "$CHECK_INTERVAL"
    continue
  fi

  # Calculate memory growth
  MEMORY_GROWTH=$((CURRENT_MEM_KB - INITIAL_MEM_KB))
  
  # Get goroutine count if possible (requires Go runtime)
  GOROUTINE_COUNT=$(curl -s http://localhost:6060/debug/pprof/goroutine 2>/dev/null | grep -c "goroutine" || echo "N/A")

  # Check if memory usage exceeds the defined threshold
  if [ "$CURRENT_MEM_KB" -gt "$MAX_MEM_KB" ]; then
    echo "🚨 MEMORY LEAK DETECTED! 🚨"
    echo "   Main PID: $SERVER_PID"
    echo "   Current Usage: $CURRENT_MEM_KB KB"
    echo "   Initial Usage: $INITIAL_MEM_KB KB"
    echo "   Growth:        $MEMORY_GROWTH KB"
    echo "   Threshold:     $MAX_MEM_KB KB"
    echo "   Goroutines:    $GOROUTINE_COUNT"
    echo "---"
    list_child_processes "$SERVER_PID"
    echo "---"
    echo "🔥 Terminating server process to prevent system instability."
    # The cleanup trap will handle the killing
    exit 1
  else
    # Log current status with more details
    printf "✅ Memory OK: %'d KB / %'d KB (Growth: %+d KB, Goroutines: %s)\n" "$CURRENT_MEM_KB" "$MAX_MEM_KB" "$MEMORY_GROWTH" "$GOROUTINE_COUNT"
    
    # Every 60 seconds, show child process tree
    if [ $((ELAPSED_TIME % 60)) -eq 0 ]; then
      list_child_processes "$SERVER_PID"
      echo ""
    fi
  fi

  sleep "$CHECK_INTERVAL"
done

echo ""
echo "📊 Final Summary:"
echo "   Runtime: $ELAPSED_TIME seconds"
echo "   Initial Memory: $INITIAL_MEM_KB KB"
echo "   Final Memory: $CURRENT_MEM_KB KB"
echo "   Total Growth: $MEMORY_GROWTH KB" 