#!/bin/bash

INTERNAL_INTERFACE=eth1
EXTERNAL_INTERFACE=eth0

VIRTUAL_IPV6=2001:db8:c0ff:ee00::/64

## 全ての設定をクリア
ip6tables -F
ip6tables -X
ip6tables -Z
ip6tables -t nat -F
ip6tables -t nat -X
ip6tables -t nat -Z
ip6tables -t mangle -F
ip6tables -t mangle -X
ip6tables -t mangle -Z
ip6tables -P INPUT ACCEPT
ip6tables -P FORWARD ACCEPT
ip6tables -P OUTPUT ACCEPT

# ループバックトラフィックを許可
ip6tables -A INPUT -i lo -j ACCEPT
ip6tables -A OUTPUT -o lo -j ACCEPT

########################################################
### General Node -> CYPHONIC Domain (Adapter Daemon) ###
########################################################
# INTERNAL_INTERFACE を介して 2001:db8:c0ff:ee00::/64 に対するパケットを開発アプリケーションで処理
ip6tables -A INPUT -i ${INTERNAL_INTERFACE} -s ${VIRTUAL_IPV6} -j ACCEPT

# EXTERNAL_INTERFACE へのフォワーディングを禁止
ip6tables -A FORWARD -i ${INTERNAL_INTERFACE} -s ${VIRTUAL_IPV6} -d ${VIRTUAL_IPV6} -j DROP

######################################
### General Node -> General Domain ###
######################################
# INTERNAL_INTERFACE を介して 2001:db8:c0ff:ee00::/64 以外に対するパケットを EXTERNAL_INTERFACE にフォワーディング
ip6tables -A INPUT -i ${INTERNAL_INTERFACE} -j ACCEPT
ip6tables -A FORWARD -i ${INTERNAL_INTERFACE} -o ${EXTERNAL_INTERFACE} -j ACCEPT

# EXTERNAL_INTERFACE にフォワーディングされるパケットの送信元IPアドレスを SNAT
ip6tables -t nat -A POSTROUTING -o ${EXTERNAL_INTERFACE} -j MASQUERADE

# EXTERNAL_INTERFACE から受信したパケットを INTERNAL_INTERFACE を介してクライアント端末に転送
ip6tables -A FORWARD -i ${EXTERNAL_INTERFACE} -o ${INTERNAL_INTERFACE} -j ACCEPT

# ステートフルなトラッキングを有効にする
ip6tables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
ip6tables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT

###################
### DNS Traffic ###
###################
# すべての DNS トラフィック（UDPおよびTCP）を許可
ip6tables -A FORWARD -p udp --sport 53 -j ACCEPT
ip6tables -A FORWARD -p udp --dport 53 -j ACCEPT
ip6tables -A FORWARD -p tcp --sport 53 -j ACCEPT
ip6tables -A FORWARD -p tcp --dport 53 -j ACCEPT

# EXTERNAL_INTERFACE を介して送信または受信される DNS トラフィックを許可
ip6tables -A FORWARD -p udp --sport 53 -o ${EXTERNAL_INTERFACE} -j ACCEPT
ip6tables -A FORWARD -p udp --dport 53 -i ${EXTERNAL_INTERFACE} -j ACCEPT
ip6tables -A FORWARD -p tcp --sport 53 -o ${EXTERNAL_INTERFACE} -j ACCEPT
ip6tables -A FORWARD -p tcp --dport 53 -i ${EXTERNAL_INTERFACE} -j ACCEPT

# EXTERNAL_INTERFACE を経由する DNS トラフィックの送信元IPアドレスを SNAT
ip6tables -t nat -A POSTROUTING -o ${EXTERNAL_INTERFACE} -p udp --dport 53 -j MASQUERADE

###################
### Save config ###
###################

# ip6tableのルールを保存
ip6tables-save >/etc/iptable/rules.v6

###############################
### Kernel packet forwarder ###
###############################

# フォーワーディングルールを適用
echo 1 > /proc/sys/net/ipv6/conf/all/forwarding


# 確認
ip6tables -nvL

#####################################################################################
# $ sudo ip6tables -nvL
#
# Chain INPUT (policy ACCEPT 1 packets, 81 bytes)
#  pkts bytes target     prot opt in     out     source                    destination
#     0     0 ACCEPT     all  --  lo     *       ::/0                      ::/0
#     0     0 ACCEPT     all  --  eth1   *       2001:db8:c0ff:ee00::/64   ::/0
#     0     0 ACCEPT     all  --  eth1   *       ::/0                      ::/0
#     0     0 ACCEPT     all  --  *      *       ::/0                      ::/0            state RELATED,ESTABLISHED

# Chain FORWARD (policy ACCEPT 0 packets, 0 bytes)
#  pkts bytes target     prot opt in     out     source                    destination
#     0     0 DROP       all  --  eth1   *       2001:db8:c0ff:ee00::/64   2001:db8:c0ff:ee00::/64
#     0     0 ACCEPT     all  --  eth1   eth0    ::/0                      ::/0
#     0     0 ACCEPT     all  --  eth0   eth1    ::/0                      ::/0
#     0     0 ACCEPT     all  --  *      *       ::/0                      ::/0            state RELATED,ESTABLISHED
#     0     0 ACCEPT     udp  --  *      *       ::/0                      ::/0            udp spt:53
#     0     0 ACCEPT     udp  --  *      *       ::/0                      ::/0            udp dpt:53
#     0     0 ACCEPT     tcp  --  *      *       ::/0                      ::/0            tcp spt:53
#     0     0 ACCEPT     tcp  --  *      *       ::/0                      ::/0            tcp dpt:53
#     0     0 ACCEPT     udp  --  *      eth0    ::/0                      ::/0            udp spt:53
#     0     0 ACCEPT     udp  --  eth0  *        ::/0                      ::/0            udp dpt:53
#     0     0 ACCEPT     tcp  --  *      eth0    ::/0                      ::/0            tcp spt:53
#     0     0 ACCEPT     tcp  --  eth0  *        ::/0                      ::/0            tcp dpt:53

# Chain OUTPUT (policy ACCEPT 0 packets, 0 bytes)
#  pkts bytes target     prot opt in     out     source                    destination
#     0     0 ACCEPT     all  --  *      lo      ::/0                      ::/0
