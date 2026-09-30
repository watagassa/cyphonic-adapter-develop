#!/bin/bash

INTERNAL_INTERFACE=eth1

# フィルターが存在する場合に削除
if tc filter show dev "$INTERNAL_INTERFACE" egress | grep -q "."; then
    tc filter del dev "$INTERNAL_INTERFACE" egress
fi

# qdiscが存在する場合に削除
if tc qdisc show dev "$INTERNAL_INTERFACE" | grep -q clsact; then
    sudo tc qdisc del dev "$INTERNAL_INTERFACE" clsact
fi

# フィルターを追加
tc qdisc add dev "$INTERNAL_INTERFACE" clsact
tc filter add dev "$INTERNAL_INTERFACE" egress bpf da obj ./scripts/ebpf/drop_icmp_error.o sec tc

# 確認
tc filter show dev "$INTERNAL_INTERFACE" egress