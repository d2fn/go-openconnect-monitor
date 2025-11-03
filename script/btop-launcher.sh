#!/usr/bin/env bash

echo "Waiting for VPN tunnel tun0 to come up..."
while ! ip route show | grep tun0; do
	sleep 1
done

btop -c $(dirname "$BASH_SOURCE[0]")/btop.config

