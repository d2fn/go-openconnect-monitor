# OpenConnect VPN Monitor


## On NixOS

First, add your Pulse VPN host and url to `config.toml`. Then you can run via the two step process

1. Start the cookie poller
```
sudo nix run .#poller
```

2. Start the openconnect process monitor
```
sudo nix run .#openconnect
```

Or use the script `launch.sh` to launch a tmux split pane showing both running processes.

