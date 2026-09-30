# Usage

```sh
### copy .env
$ cp ./.docker/.env{.sample,}

### change args: create a bridge interface (br0) and bind it to an existing interface.
$ vim ./.docker/adapter/scripts/external-network.sh
~~~
BRIDGE_INTERFACE_NAME=enx00e04c20062e
EXTERNAL_NETWORK_SUBNET=192.168.11.0/24
BRIDGE_IPV4_ADDRESS=192.168.11.30
~~~

### build the environment: First time
$ make init

### build the environment: 2nd time onwards
$ make up
```
