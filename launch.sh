#!/usr/bin/env bash

tmux new-session -d -s openconnect-pulse -n openconnect 'nix run .#poller'
tmux split-window -v -t openconnect-pulse:0 'sudo nix run .#openconnect'
tmux attach -t openconnect-pulse

