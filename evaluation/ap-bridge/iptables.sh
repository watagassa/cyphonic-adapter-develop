#!/bin/bash

iptables -A FORWARD -i eth0 -o wlan0 -j ACCEPT
iptables -A FORWARD -i wlan0 -o eth0 -m state --state RELATED,ESTABLISHED -j ACCEPT
iptables -t nat -A POSTROUTING -o wlan0 -j MASQUERADE

iptables-save >/etc/iptables/rules.v4

echo 1 >/proc/sys/net/ipv4/ip_forward

iptables -t nat -vL --line-numbers
