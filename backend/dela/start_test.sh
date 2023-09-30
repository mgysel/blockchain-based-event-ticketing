#!/bin/sh

# This script creates a new tmux session and starts nodes according to the
# instructions is README.md. The test session can be kill with kill_test.sh.

set -o errexit

command -v tmux >/dev/null 2>&1 || { echo >&2 "tmux is not on your PATH!"; exit 1; }

# Launch session
s="d-voting-test1"
# Setup initial nodes/ports
PORT1=2041
NODE1=20

NODE2=`expr $NODE1 + 1`
NODE3=`expr $NODE1 + 2`
PORT2=`expr $PORT1 + 1`
PORT3=`expr $PORT1 + 2`

tmux list-sessions | rg "^$s:" >/dev/null 2>&1 && { echo >&2 "A session with the name $s already exists; kill it and try again"; exit 1; }

tmux new -s $s -d

tmux split-window -t $s -h
tmux split-window -t $s:0.%0
tmux split-window -t $s:0.%1

# session s, window 0, panes 0 to 2
master="tmux send-keys -t $s:0.%0"
node1="tmux send-keys -t $s:0.%1"
node2="tmux send-keys -t $s:0.%2"
node3="tmux send-keys -t $s:0.%3"

pk=adbacd10fdb9822c71025d6d00092b8a4abb5ebcb673d28d863f7c7c5adaddf3

$node1 "LLVL=info memcoin --config /tmp/node${NODE1} start --listen tcp://127.0.0.1:${PORT1}" C-m
$node2 "LLVL=info memcoin --config /tmp/node${NODE2} start --listen tcp://127.0.0.1:${PORT2}" C-m
$node3 "LLVL=info memcoin --config /tmp/node${NODE3} start --listen tcp://127.0.0.1:${PORT3}" C-m


tmux a
