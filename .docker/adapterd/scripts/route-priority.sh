#!/bin/bash

TARGET='precedence ::ffff:0:0\/96  100'

sed -i -e "s/#\(${TARGET}\)/\1/g" /etc/gai.conf
