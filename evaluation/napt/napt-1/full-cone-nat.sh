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

OUTPUT_IP="10.0.100.105"
OUTPUT_INTERFACE="eth0"

## Node 1
INNER_IP_1="192.168.10.5"
iptables -t nat -A PREROUTING -i ${OUTPUT_INTERFACE} -j DNAT --to-destination ${INNER_IP_1}
iptables -t nat -A POSTROUTING -o ${OUTPUT_INTERFACE} -j SNAT --to-source ${OUTPUT_IP}

## Node 2
INNER_IP_2="192.168.10.6"
iptables -t nat -A PREROUTING -i ${OUTPUT_INTERFACE} -j DNAT --to-destination ${INNER_IP_2}
iptables -t nat -A POSTROUTING -o ${OUTPUT_INTERFACE} -j SNAT --to-source ${OUTPUT_IP}

iptables-save >/etc/iptables/rules.v4

iptables -t nat -vL --line-numbers
