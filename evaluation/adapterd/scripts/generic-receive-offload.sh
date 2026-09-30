#!/bin/sh

INTERNAL_INTERFACE=eth1

ethtool -K ${INTERNAL_INTERFACE} gro off
ethtool -k ${INTERNAL_INTERFACE} | grep generic-receive-offload
