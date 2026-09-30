# CockroachDB

### インストール

```shell
### リポジトリを更新
$ sudo apt update && sudo apt upgrade -y

### CockroachDB インストール
$ curl https://binaries.cockroachdb.com/cockroach-v23.1.8.linux-amd64.tgz | tar -xz && sudo cp -i cockroach-v23.1.8.linux-amd64/cockroach /usr/local/bin/

### 確認
$ cockroach version
Build Tag:        v23.1.8
Build Time:       2023/08/04 18:11:44
Distribution:     CCL
Platform:         linux amd64 (x86_64-pc-linux-gnu)
Go Version:       go1.19.10
C Compiler:       gcc 6.5.0
Build Commit ID:  2c1d4afd2c2de091a00a4dba4392ceeaf8504a95
Build Type:       release
Enabled Assertions: false
```

### クラスタ構築

```shell
### シングルクラスタ Insecure モードで起動
$ sudo cockroach start-single-node --background --insecure --store=/var/lib/cockroach/data && init --host=10.0.100.204:26257 --insecure
*
* WARNING: ALL SECURITY CONTROLS HAVE BEEN DISABLED!
*
* This mode is intended for non-production testing only.
*
* In this mode:
* - Your cluster is open to any client that can access any of your IP addresses.
* - Intruders with access to your machine or network can observe client-server traffic.
* - Intruders can log in without password and read or write any data in the cluster.
* - Intruders can consume all your server's resources and cause unavailability.
*
*
* INFO: To start a secure server without mandating TLS for clients,
* consider --accept-sql-without-tls instead. For other options, see:
*
* - https://go.crdb.dev/issue-v/53404/v23.1
* - https://www.cockroachlabs.com/docs/v23.1/secure-a-cluster.html
*
*
* WARNING: Running a server without --sql-addr, with a combined RPC/SQL listener, is deprecated.
* This feature will be removed in a later version of CockroachDB.
*
*
* WARNING: neither --listen-addr nor --advertise-addr was specified.
* The server will advertise "cyphonic-cockroach" to other nodes, is this routable?
*
* Consider using:
* - for local-only servers:  --listen-addr=localhost:36257 --sql-addr=localhost:26257
* - for multi-node clusters: --listen-addr=:36257 --sql-addr=:26257 --advertise-addr=<host/IP addr>
*
*
init: unrecognized option '--host=10.0.100.204:26257'

### 接続
$ cockroach sql --insecure
```

# データベースの作成

```shell
### データベースとユーザを作成
> CREATE DATABASE IF NOT EXISTS cyphonic ENCODING = 'UTF-8';
> CREATE USER cyphonic;
> GRANT ALL ON ALL TABLES IN SCHEMA public TO cyphonic;

### 確認
> show databases;
  database_name | owner | primary_region | secondary_region | regions | survival_goal
----------------+-------+----------------+------------------+---------+----------------
  cyphonic      | root  | NULL           | NULL             | {}      | NULL
  defaultdb     | root  | NULL           | NULL             | {}      | NULL
  postgres      | root  | NULL           | NULL             | {}      | NULL
  system        | node  | NULL           | NULL             | {}      | NULL
(4 rows)

> SHOW USERS;
  username | options | member_of
-----------+---------+------------
  admin    |         | {}
  cyphonic |         | {}
  root     |         | {admin}
(3 rows)

> SELECT * FROM pg_catalog.pg_user;
  usename  |  usesysid  | usecreatedb | usesuper | userepl | usebypassrls |  passwd  | valuntil | useconfig
-----------+------------+-------------+----------+---------+--------------+----------+----------+------------
  cyphonic | 1592320289 |      f      |    f     |    f    |      f       | ******** | NULL     | NULL
  root     | 1546506610 |      t      |    t     |    f    |      f       | ******** | NULL     | NULL
  node     | 3233629770 |      t      |    t     |    f    |      f       | ******** | NULL     | NULL
(3 rows)
```

# スキーマの作成

```shell
$ cockroach sql --insecure -u root --database=cyphonic -p 26257
#
# Welcome to the CockroachDB SQL shell.
# All statements must be terminated by a semicolon.
# To exit, type: \q.
#
# Server version: CockroachDB CCL v23.1.8 (x86_64-pc-linux-gnu, built 2023/08/04 18:11:44, go1.19.10) (same version as client)
# Cluster ID: 326632f9-b586-4e18-a2cb-8956c58934d8
#
# Enter \? for a brief introduction.
#

> \i /home/vagrant/init.db/000-cyphonic-ddl.sql;
> \i /home/vagrant/init.db/001-administrative-dml.sql;
> \i /home/vagrant/init.db/002-sample-info-dml.sql;
> \i /home/vagrant/init.db/003-adapter-sample-dml.sql;
```

# その他のコマンド

```shell
> DROP USER cyphonic;
> DROP DATABASE IF EXISTS cyphonic CASCADE;
```
