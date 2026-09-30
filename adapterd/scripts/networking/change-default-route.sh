#!/bin/sh

INTERFACE=wlan0
DEFAULT_GATEWAY_ADDR=10.0.3.254

route delete default
route add default gw ${DEFAULT_GATEWAY_ADDR} ${INTERFACE}
ip route
