#!/bin/bash

INTERNAL_INTERFACE=eth1
EXTERNAL_INTERFACE=eth0

VIRTUAL_IPV4=198.18.0.0/16

## 全ての設定をクリア
iptables -F
iptables -X
iptables -Z
iptables -t nat -F
iptables -t nat -X
iptables -t nat -Z
iptables -t mangle -F
iptables -t mangle -X
iptables -t mangle -Z
iptables -P INPUT ACCEPT
iptables -P FORWARD ACCEPT
iptables -P OUTPUT ACCEPT

# ループバックトラフィックを許可
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

########################################################
### General Node -> CYPHONIC Domain (Adapter Daemon) ###
########################################################
# INTERNAL_INTERFACE を介して 198.18.0.0/16 に対するパケットを開発アプリケーションで処理
iptables -A INPUT -i ${INTERNAL_INTERFACE} -s ${VIRTUAL_IPV4} -j ACCEPT

# EXTERNAL_INTERFACE へのフォワーディングを禁止
iptables -A FORWARD -i ${INTERNAL_INTERFACE} -s ${VIRTUAL_IPV4} -d ${VIRTUAL_IPV4} -j DROP

######################################
### General Node -> General Domain ###
######################################
# INTERNAL_INTERFACE を介して 198.18.0.0/16 以外に対するパケットを EXTERNAL_INTERFACE にフォワーディング
iptables -A INPUT -i ${INTERNAL_INTERFACE} -j ACCEPT
iptables -A FORWARD -i ${INTERNAL_INTERFACE} -o ${EXTERNAL_INTERFACE} -j ACCEPT

# EXTERNAL_INTERFACE にフォワーディングされるパケットの送信元IPアドレスを SNAT
iptables -t nat -A POSTROUTING -o ${EXTERNAL_INTERFACE} -j MASQUERADE

# EXTERNAL_INTERFACE から受信したパケットを INTERNAL_INTERFACE を介してクライアント端末に転送
iptables -A FORWARD -i ${EXTERNAL_INTERFACE} -o ${INTERNAL_INTERFACE} -j ACCEPT

# ステートフルなトラッキングを有効にする
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT

###################
### DNS Traffic ###
###################
# すべての DNS トラフィック（UDPおよびTCP）を許可
iptables -A FORWARD -p udp --sport 53 -j ACCEPT
iptables -A FORWARD -p udp --dport 53 -j ACCEPT
iptables -A FORWARD -p tcp --sport 53 -j ACCEPT
iptables -A FORWARD -p tcp --dport 53 -j ACCEPT

# EXTERNAL_INTERFACE を介して送信または受信される DNS トラフィックを許可
iptables -A FORWARD -p udp --sport 53 -o ${EXTERNAL_INTERFACE} -j ACCEPT
iptables -A FORWARD -p udp --dport 53 -i ${EXTERNAL_INTERFACE} -j ACCEPT
iptables -A FORWARD -p tcp --sport 53 -o ${EXTERNAL_INTERFACE} -j ACCEPT
iptables -A FORWARD -p tcp --dport 53 -i ${EXTERNAL_INTERFACE} -j ACCEPT

# EXTERNAL_INTERFACE を経由する DNS トラフィックの送信元IPアドレスを SNAT
iptables -t nat -A POSTROUTING -o ${EXTERNAL_INTERFACE} -p udp --dport 53 -j MASQUERADE

###################
### Save config ###
###################

# iptablesのルールを保存
iptables-save >/etc/iptables/rules.v4

###############################
### Kernel packet forwarder ###
###############################

# フォーワーディングルールを適用
echo 1 >/proc/sys/net/ipv4/ip_forward

# 確認
iptables -nvL

#####################################################################################
# $ sudo iptables -nvL
#
# Chain INPUT (policy ACCEPT 1 packets, 81 bytes)
#  pkts bytes target     prot opt in     out     source               destination
#     0     0 ACCEPT     all  --  lo     *       0.0.0.0/0            0.0.0.0/0
#     0     0 ACCEPT     all  --  eth1   *       198.18.0.0/16        0.0.0.0/0
#     0     0 ACCEPT     all  --  eth1   *       0.0.0.0/0            0.0.0.0/0
#     0     0 ACCEPT     all  --  *      *       0.0.0.0/0            0.0.0.0/0            state RELATED,ESTABLISHED

# Chain FORWARD (policy ACCEPT 0 packets, 0 bytes)
#  pkts bytes target     prot opt in     out     source               destination
#     0     0 DROP       all  --  eth1   *       198.18.0.0/16        198.18.0.0/16
#     0     0 ACCEPT     all  --  eth1   eth0    0.0.0.0/0            0.0.0.0/0
#     0     0 ACCEPT     all  --  eth0   eth1    0.0.0.0/0            0.0.0.0/0
#     0     0 ACCEPT     all  --  *      *       0.0.0.0/0            0.0.0.0/0            state RELATED,ESTABLISHED
#     0     0 ACCEPT     udp  --  *      *       0.0.0.0/0            0.0.0.0/0            udp spt:53
#     0     0 ACCEPT     udp  --  *      *       0.0.0.0/0            0.0.0.0/0            udp dpt:53
#     0     0 ACCEPT     tcp  --  *      *       0.0.0.0/0            0.0.0.0/0            tcp spt:53
#     0     0 ACCEPT     tcp  --  *      *       0.0.0.0/0            0.0.0.0/0            tcp dpt:53
#     0     0 ACCEPT     udp  --  *      eth0    0.0.0.0/0            0.0.0.0/0            udp spt:53
#     0     0 ACCEPT     udp  --  eth0  *        0.0.0.0/0            0.0.0.0/0            udp dpt:53
#     0     0 ACCEPT     tcp  --  *      eth0    0.0.0.0/0            0.0.0.0/0            tcp spt:53
#     0     0 ACCEPT     tcp  --  eth0  *        0.0.0.0/0            0.0.0.0/0            tcp dpt:53

# Chain OUTPUT (policy ACCEPT 0 packets, 0 bytes)
#  pkts bytes target     prot opt in     out     source               destination
#     0     0 ACCEPT     all  --  *      lo      0.0.0.0/0            0.0.0.0/0
