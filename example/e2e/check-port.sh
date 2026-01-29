#!/bin/bash

set -x -e

ADDR=$1
if [ -z "$ADDR" ]; then
	echo "usage: $0 ADDRESS [TIMES] [SLEEP] [PIDFILE]"
	echo "address required"
	exit 2
fi
TIMES=${2:-30}
SLEEP=${3:-1}
PIDFILE=${4:-}

# Check if process from PID file is still running
# Returns 0 if no pidfile specified, process is running, or pidfile doesn't exist yet
# Returns 1 if pidfile exists but process is dead
check_pid() {
	if [ -z "$PIDFILE" ]; then
		return 0
	fi
	if [ ! -f "$PIDFILE" ]; then
		# PID file doesn't exist yet - service may still be starting
		return 0
	fi
	local pid
	pid=$(cat "$PIDFILE" 2>/dev/null)
	if [ -z "$pid" ]; then
		return 0
	fi
	if kill -0 "$pid" 2>/dev/null; then
		return 0
	fi
	echo "process $pid from $PIDFILE is no longer running"
	return 1
}

ERROR=1

set +x
for i in $(seq $TIMES); do
	sleep "$SLEEP"

	# Early exit if monitored process died
	if ! check_pid; then
		echo "service process terminated before becoming ready"
		exit 3
	fi

	set +e
	curl "$ADDR" &>/dev/null
	ERROR=$?
	set -e
	if [ $ERROR -eq 0 ]; then
		break
	fi
done

# Final PID check - ensure process is still alive even if curl succeeded
if [ $ERROR -eq 0 ] && ! check_pid; then
	echo "service process terminated"
	exit 3
fi

exit $ERROR
