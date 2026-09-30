#!/bin/bash

iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X
iptables -P INPUT ACCEPT
iptables -P FORWARD ACCEPT
iptables -P OUTPUT ACCEPT

OUTPUT_INTERFACE="eth0"

iptables -t nat -A POSTROUTING -o $OUTPUT_INTERFACE -j MASQUERADE --random

iptables-save >/etc/iptables/rules.v4

iptables -t nat -vL --line-numbers
