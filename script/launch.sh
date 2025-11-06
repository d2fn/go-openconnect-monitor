#!/usr/bin/env bash

session_name="openconnect-pulse"

tmux new-session  -d -s $session_name -n btop-tun0 'script/btop-launcher.sh'
tmux split-window -v -t $session_name:1 'nix run .#poller'
tmux split-window -v -t $session_name:1 'sudo nix run .#openconnect'
tmux attach -t $session_name

