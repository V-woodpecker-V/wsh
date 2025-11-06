#!/bin/bash
# Demo plugin for wsh

# During plugin loading, only register and exit
# TODO: Remove WSH_BINARY check once we ensure wsh is always in PATH during plugin load
if [ -n "$WSH_BINARY" ]; then
    $WSH_BINARY args --register \
        -D --demo "Demo plugin for testing" \
        -m --message "msg" "Message to display" \
        -v --verbose "Enable verbose output"
    exit 0
fi

# Plugin execution logic (when called by wsh -D)
message="${message:-Hello from demo plugin!}"

if [ -n "$verbose" ]; then
    echo "=== Demo Plugin ===" >&2
    echo "Message: $message" >&2
    echo "Verbose: enabled" >&2
    echo "==================" >&2
fi

echo "$message"
