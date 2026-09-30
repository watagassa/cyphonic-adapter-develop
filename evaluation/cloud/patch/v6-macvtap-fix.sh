#!/bin/bash

AS_ADDRESS=fde4:db8::201/64
NMS_ADDRESS=fde4:db8::202/64
CONTROLLER_ADDRESS=fde4:db8::205/64

AS_TAP_NAME=macvtap3
NMS_TAP_NAME=macvtap1
CONTROLLER_TAP_NAME=macvtap0

## as
ip addr add ${AS_ADDRESS} dev ${AS_TAP_NAME}

## nms
ip addr add ${NMS_ADDRESS} dev ${NMS_TAP_NAME}

## controller
ip addr add ${CONTROLLER_ADDRESS} dev ${CONTROLLER_TAP_NAME}
