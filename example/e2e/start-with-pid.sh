#!/bin/bash

# Starts a command in the background and writes its PID to a file.
# Useful for monitoring service startup with check-port.sh
#
# Usage: start-with-pid.sh PIDFILE COMMAND [ARGS...]
#
# Example:
#   ./start-with-pid.sh /tmp/myservice.pid ./myservice --port 8080
#   ./check-port.sh http://localhost:8080 30 1 /tmp/myservice.pid

set -e

PIDFILE=$1
if [ -z "$PIDFILE" ]; then
	echo "usage: $0 PIDFILE COMMAND [ARGS...]"
	echo "pidfile path required"
	exit 2
fi
shift

if [ $# -eq 0 ]; then
	echo "usage: $0 PIDFILE COMMAND [ARGS...]"
	echo "command required"
	exit 2
fi

# Ensure parent directory exists
PIDDIR=$(dirname "$PIDFILE")
if [ ! -d "$PIDDIR" ]; then
	mkdir -p "$PIDDIR"
fi

# Remove stale PID file if it exists
rm -f "$PIDFILE"

# Start the command in the background
"$@" &
PID=$!

# Write PID to file
echo "$PID" > "$PIDFILE"

echo "started process $PID, pidfile: $PIDFILE"
echo "command: $*"
