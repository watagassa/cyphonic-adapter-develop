# CYPHONIC Adapter Evaluation and Demonstration

- [IEEE CCNC2024](https://ccnc2024.ieee-ccnc.org/program/demos) で実施したデモンストレーションにおける環境構築情報

> CYPHONIC Adapter: Enabling Secure P2P Connectivity for Diverse IoT Devices

> Ren Goto; Kazushige Matama; Ryouta Aihata; Michiyo Suda; Hidekazu Suzuki; Katsuhiro Naito

## 検証機材

### PC・マイコン

| 機材                                                                                                                   | 個数 |
| :--------------------------------------------------------------------------------------------------------------------- | ---: |
| Lenovo ThinkPad E14 Gen3                                                                                               |    2 |
| Lenovo ThinkPad E14 Gen3 充電器 65W                                                                                    |    2 |
| Raspberry Pi 4 model B（napt-1, napt-2, adapter-v4-1, adapter-v6-1, prd-trs, prd-general-1, prd-general-2, ap-bridge） |    8 |
| TP-Link IoT Camera VIGI C450                                                                                           |    1 |
| TP-Link IoT Camera VIGI C540V                                                                                          |    1 |

### ネットワーク機器

| 機材                                        | 個数 |
| :------------------------------------------ | ---: |
| TP-Link Router ER605                        |    1 |
| TP-Link Router ER605 9V 0.85A 電源          |    1 |
| TP-Link L2 Switch TL-SG1005P                |    1 |
| TP-Link L2 Switch TL-SG1005P 48V 1.25A 電源 |    1 |
| Anker USB ハブ                              |    2 |
| SANWA SUPPLY PoE アダプタ                   |    2 |
| USB-Ether                                   |    4 |
| LAN ケーブル 赤                             |    1 |
| LAN ケーブル 緑                             |    1 |
| LAN ケーブル 黄                             |    1 |
| LAN ケーブル 橙                             |    2 |
| LAN ケーブル 青                             |    4 |
| LAN ケーブル 白                             |    5 |
| USB Type-C                                  |    9 |
| USB Type-C（青）                            |    1 |

### その他周辺機器

| 機材                             | 個数 |
| :------------------------------- | ---: |
| モバイルモニター（私物）         |    1 |
| キーボード（私物）               |    1 |
| マウス（私物）                   |    1 |
| Raspberrpy 4 用 micro HDMI       |    1 |
| HDMI（私物）                     |    1 |
| 電源タップ 7 口                  |    1 |
| 電源タップ 4 個口 スイングプラグ |    1 |
| 電源 3Pin → 2Ping 変換アダプタ   |    1 |
| ジャンパ線                       |    1 |

## バージョン情報

- CYPHONIC クラウド / CYPHONIC ノード（2023.12.30 latest）
  - https://github.com/Pluslab/cyphonic/tree/d8cd7d9cebf853009369ae0b19e01421355a308a
- CYPHONIC アダプタ（2023.12.30 latest）
  - https://github.com/Pluslab/cyphonic-adapter/tree/e0fe400461399838380af1873ae2a78e013bfb8e

## 検証機器情報

- ER605 Router：[http://10.0.100.1/webpages/index.html](http://10.0.100.1/webpages/index.html)

|     | ER605 Link IPv4 (Global)            | ER605 Link IPv6 (Global) | NAPT Private IPv4        | Backup Link (Wi-Fi)   | Host name          |                           CPU | Memory | OS                  | Functions                      |
| :-: | :---------------------------------- | :----------------------- | :----------------------- | :-------------------- | :----------------- | ----------------------------: | -----: | :------------------ | :----------------------------- |
|  1  | 10.0.100.1/24 (LAN)                 | -                        | -                        | 192.168.21.3/24 (WAN) | ER605              |                             - |      - | -                   | L3 Switch (Router)             |
|  2  | 192.168.21.1/24 (eth0 -> ER605 WAN) | -                        | -                        | ◯ (FROM stepping)     | ap-bridge          | 4 cores, 4 threads @ 1.50 GHz |  4 GiB | Debian GNU/Linux 11 | AP Bridge (Raspberry Pi 4)     |
|  3  | 10.0.100.200/24 (enp2s0)            | -                        | -                        | - (FROM stepping)     | laptop-uguisu      | 4 cores, 8 threads @ 2.60 GHz | 16 GiB | Ubuntu 22.04        | ThinkPad (VM Host)             |
|  4  | 10.0.100.201/24 (eth1)              | fde4:db8::201/64 (eth1)  | -                        | -                     | as                 | 1 cores, 1 threads @ 2.60 GHz |  1 GiB | Ubuntu 20.04        | asd (VM)                       |
|  5  | 10.0.100.202/24 (eth1)              | fde4:db8::202/64 (eth1)  | -                        | -                     | nms01              | 1 cores, 1 threads @ 2.60 GHz |  1 GiB | Ubuntu 20.04        | nmsd (VM)                      |
|  6  | 10.0.100.203/24 (eth0)              | fde4:db8::203/64 (eth0)  | -                        | - (FROM stepping)     | prd-trs            | 4 cores, 4 threads @ 1.50 GHz |  8 GiB | Debian GNU/Linux 11 | trsd (Raspberry Pi 4)          |
|  7  | 10.0.100.204/24 (eth1)              | -                        | -                        | -                     | cyphonic-cockroach | 1 cores, 1 threads @ 2.60 GHz |  1 GiB | Ubuntu 20.04        | db (VM)                        |
|  8  | 10.0.100.205/24 (eth1)              | fde4:db8::205/64 (eth1)  | -                        | -                     | controller         | 1 cores, 1 threads @ 2.60 GHz |  1 GiB | Ubuntu 20.04        | controller (VM)                |
|  9  | 10.0.100.105/24 (eth0)              | -                        | 192.168.10.1/24 (eth1)   | - (FROM stepping)     | napt-1             | 4 cores, 4 threads @ 1.50 GHz |  4 GiB | Debian GNU/Linux 11 | Full Cone NAT (Raspberry Pi 4) |
| 10  | 10.0.100.106/24 (eth0)              | -                        | 192.168.20.1/24 (eth1)   | - (FROM stepping)     | napt-2             | 4 cores, 4 threads @ 1.50 GHz |  4 GiB | Debian GNU/Linux 11 | Full Cone NAT (Raspberry Pi 4) |
| 11  | -                                   | -                        | 192.168.10.5/24 (eth0)   | - (FROM napt-1)       | adapter-v4-1       | 4 cores, 4 threads @ 1.50 GHz |  8 GiB | Debian GNU/Linux 11 | CYPHONIC Adapter with IPv4     |
| 12  | -                                   | fde4:db8::a2/64 (eth0)   | -                        | ◯ (FROM stepping)     | adapter-v6-1       | 4 cores, 4 threads @ 1.50 GHz |  8 GiB | Debian GNU/Linux 11 | CYPHONIC Adapter with IPv6     |
| 13  | -                                   | -                        | 192.168.20.5/24 (enp2s0) | - (FROM napt-2)       | laptop-asagi       | 4 cores, 8 threads @ 2.60 GHz | 16 GiB | Ubuntu 22.04        | ThinkPad (CYPHONIC Node)       |

### CYPHONIC アダプタ

|     | Device Name         | Account ID          | Device ID                                                | FQDN                                                                              | Node ID                          | Adapter Network |
| :-: | :------------------ | :------------------ | :------------------------------------------------------- | :-------------------------------------------------------------------------------- | :------------------------------- | :-------------: |
|  1  | cyphonic-adapter-01 | test10@cyphonic.org | 2bac8cfcfa0a0e8fe7655cff0fec896866eff52e94efce030be15337 | 2bac8cfcfa0a0e8fe7655cff0fec896866eff52e94efce030be15337.nms01.local.cyphonic.org | c8032745c1e75aae843896b7dfc42617 |      IPv4       |
|  2  | cyphonic-adapter-02 | test10@cyphonic.org | ca4698e4ba6b2ff74fcab04584221d8a1534613707493ab51d0084b2 | ca4698e4ba6b2ff74fcab04584221d8a1534613707493ab51d0084b2.nms01.local.cyphonic.org | 1724308be3aa5f9489ab5dd82dbd543e |      IPv6       |

### 一般ノード

|     | Device Name         | MAC Address       | Account ID          | Device ID                                                | FQDN                                                                              | Node ID                          | Adapter ID (Device ID)                                   |
| :-: | :------------------ | :---------------- | :------------------ | :------------------------------------------------------- | :-------------------------------------------------------------------------------- | :------------------------------- | :------------------------------------------------------- |
|  1  | Camera-VIGI-C450-1  | 5c:62:8b:7f:c9:9f | test10@cyphonic.org | 61df7869c6ea4fca675a6272d25e80cb841d0c4ce1b2b2b7e0815bb2 | 61df7869c6ea4fca675a6272d25e80cb841d0c4ce1b2b2b7e0815bb2.nms01.local.cyphonic.org | 84707019599d56b1b47fe066c0f7bf3c | 2bac8cfcfa0a0e8fe7655cff0fec896866eff52e94efce030be15337 |
|  2  | Camera-VIGI-C540V-1 | f0:a7:31:19:1a:fb | test10@cyphonic.org | 395c73de6723d246159f3c41160363da996038a34145a4ee8752a0aa | 395c73de6723d246159f3c41160363da996038a34145a4ee8752a0aa.nms01.local.cyphonic.org | 8d787a6849f15d68bf2dce5f868a0e63 | ca4698e4ba6b2ff74fcab04584221d8a1534613707493ab51d0084b2 |
|  3  | prd-general-1       | dc:a6:32:bf:8b:c8 | test10@cyphonic.org | 3ef68dac78ec4bce91d2cc31219e4f7b3911148453d1f14f7fc9f701 | 3ef68dac78ec4bce91d2cc31219e4f7b3911148453d1f14f7fc9f701.nms01.local.cyphonic.org | aeb83ebc938656dcba43c3087ca903da | 2bac8cfcfa0a0e8fe7655cff0fec896866eff52e94efce030be15337 |
|  4  | prd-general-2       | dc:a6:32:90:9d:1b | test10@cyphonic.org | 4f0b1fb5663eb66f0a5155606f91e37f6333263716df86f451a273c0 | 4f0b1fb5663eb66f0a5155606f91e37f6333263716df86f451a273c0.nms01.local.cyphonic.org | d85d2ff0c23b51259c1f6aba53c21730 | ca4698e4ba6b2ff74fcab04584221d8a1534613707493ab51d0084b2 |

### CYPHONIC ノード

|     | Device Name | Account ID          | Device ID                                                | FQDN                                                                              | Node ID                          |
| :-: | :---------- | :------------------ | :------------------------------------------------------- | :-------------------------------------------------------------------------------- | :------------------------------- |
|  1  | node        | test01@cyphonic.org | f6efb4e0355cae5a37ddd58a286d4bcbe6ff4ca585c67cb02f263aa3 | f6efb4e0355cae5a37ddd58a286d4bcbe6ff4ca585c67cb02f263aa3.nms01.local.cyphonic.org | a370e86e3e205da1a7c99de5c783cfba |
|  2  | node2       | test02@cyphonic.org | f7b8c180a4d573cb55479f76377f35fbbe58453362736f6640c2612a | f7b8c180a4d573cb55479f76377f35fbbe58453362736f6640c2612a.nms01.local.cyphonic.org | 40ead214adba5d3b9db3cbafbbda56f6 |

## デモンストレーション手順書

### 環境構築：CYPHONIC クラウド

```shell
### uguisu に入り Vagrant 起動
$ ssh uguisu

### Vagrant 再起動
$ cd ccnc2
$ vagrant up

### macvtap 修正
$ sudo ./v6-macvtap-fix.sh

### adapter-v6-1 から以下の IPv6 アドレスに疎通確認
$ ping fde4:db8::202
$ ping fde4:db8::203

### CockroachDB 再起動
$ sudo cockroach start-single-node --background --insecure --store=/var/lib/cockroach/data && init --host=10.0.100.204:26257 --insecure
```

### 環境構築：CYPHONIC ノード

```shell
### systemd-resolved プロセスを kill
$ sudo kill -9 $(lsof -t -i:53)

### CoreDNS 起動
$ sudo make coredns/run.v4 &

### Noded 起動
$ sudo make run/password
```

## 環境構築：CYPHONIC アダプタ

```shell
### CoreDNS 起動
$ sudo make coredns/run.v4 &

### GRO OFF
$ sudo ./generic-receive-offload.sh

### Adapterd 起動
$ sudo make run/password
```

## 【メモ】static IPv6 の追加

```shell
### sysctl.conf ファイルを編集
$ sudo vim /etc/sysctl.conf
net.ipv6.conf.all.disable_ipv6 = 0
net.ipv6.conf.default.disable_ipv6 = 0

### 適用
$ sudo sysctl -p

### 確認
$ cat /proc/sys/net/ipv6/conf/all/disable_ipv6
0 <--- 有効化されている
```

```shell
### 50-vagrant.yaml ファイルを編集
$ sudo vim /etc/netplan/50-vagrant.yaml
```

```yaml
---
network:
  version: 2
  renderer: networkd
  ethernets:
    eth1:
      addresses:
        - 10.0.100.211/24
        - fde4:db8::211/64 ## 追加
```

```shell
### 適用
$ sudo netplan apply
```
