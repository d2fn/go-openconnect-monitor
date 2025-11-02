#!/usr/bin/env bash

session_name="openconnect-pulse"

tmux new-session -d -s $session_name -n openconnect 'btop -c btop.config'
tmux split-window -v -t $session_name:0 'nix run .#poller'
tmux split-window -v -t $session_name:0 'sudo nix run .#openconnect'
tmux attach -t $session_name

