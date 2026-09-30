#!/bin/sh

BRIDGE_INTERFACE_NAME=enx00e04c20062e
EXTERNAL_NETWORK_SUBNET=192.168.11.0/24
BRIDGE_IPV4_ADDRESS=192.168.11.30

docker network create --driver=bridge --subnet=${EXTERNAL_NETWORK_SUBNET} --gateway=${BRIDGE_IPV4_ADDRESS} --opt "com.docker.network.bridge.name"="br0" adapter-external
sudo nmcli con add type bridge-slave ifname ${BRIDGE_INTERFACE_NAME} master br0
sudo nmcli con up bridge-slave-${BRIDGE_INTERFACE_NAME}
sudo nmcli con show
