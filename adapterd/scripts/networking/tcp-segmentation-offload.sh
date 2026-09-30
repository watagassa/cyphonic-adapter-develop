#!/bin/sh

INTERNAL_INTERFACE=eth1

ethtool -K ${INTERNAL_INTERFACE} tx off
ethtool -k ${INTERNAL_INTERFACE} | grep tcp-segmentation-offload
