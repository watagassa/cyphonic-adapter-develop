#!/bin/bash

P_CPU=$(fgrep 'physical id' /proc/cpuinfo | sort -u | wc -l)
CORES=$(fgrep 'cpu cores' /proc/cpuinfo | sort -u | sed 's/.*: //')
L_PRC=$(fgrep 'processor' /proc/cpuinfo | wc -l)
H_TRD=$((L_PRC / P_CPU / CORES))

echo -n "${L_PRC} processer"
[ "${L_PRC}" -ne 1 ] && echo -n "s"
echo -n " = ${P_CPU} socket"
[ "${P_CPU}" -ne 1 ] && echo -n "s"
echo -n " x ${CORES} core"
[ "${CORES}" -ne 1 ] && echo -n "s"
echo -n " x ${H_TRD} thread"
[ "${H_TRD}" -ne 1 ] && echo -n "s"
echo
