#!/bin/bash

set -x -e

ADDR=$1
if [ -z "$ADDR" ]; then
	echo "usage: $0 ADDRESS [TIMES] [SLEEP]"
	echo "address required"
	exit 2
fi
TIMES=${2:-30}
SLEEP=${3:-1}

LOOPS=0
ERROR=1

set +x
for i in $(seq $TIMES); do
	sleep 1
	set +e
	curl $1 &>/dev/null
	ERROR=$?
	set -e
	if [ $ERROR -eq 0 ]; then
		break
	fi
done 

exit $ERROR
