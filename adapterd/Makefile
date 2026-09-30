GOCMD=go
GORUN=$(GOCMD) run
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GODOC=$(GOCMD)doc
GOLANGCI=golangci-lint

ADAPTER_HOSTUSER=adapter
ADAPTER_ENDPOINT=10.0.3.230



# runs
#===============================================================
run/version: ## check CYPHONIC adapter status
	@$(GORUN) ./cmd/adapterd version

run/password: ## start CYPHONIC adapter
	@$(GORUN) ./cmd/adapterd start

.PHONY: coredns/init
coredns/init: ## Install CoreDNS
	./scripts/dns/coredns.sh

.PHONY: coredns/run.v4
coredns/run.v4: ## Run CoreDNS in v4 mode
	coredns -conf ./scripts/dns/Corefile.v4

.PHONY: coredns/run
coredns/run: ## Run CoreDNS in v6 mode
	coredns -conf ./scripts/dns/Corefile

.PHONY: node_exporter/init
node_exporter/init: ## Install node_exporter
	./scripts/exporter/node_exporter.sh

.PHONY: node_exporter/run
node_exporter/run: ## Run node_exporter
	node_exporter

.PHONY: redis/init
redis/init: ## Install redis-server
	@sudo ./scripts/redis/redis.sh

.PHONY: redis/run
redis/run: ## Start redis-server service
	@sudo systemctl restart redis-server
	@sudo systemctl --no-pager status redis-server

.PHONY: iproute2/init
iproute2/init: ## Install iproute2
	@sudo ./scripts/ebpf/iproute2.sh

.PHONY: run/setup
run/setup: gro-off enable-ipv4 enable-ipv6 tc-on ## Run setup in normal mode
	@echo "Normal mode setup completed"

.PHONY: run/setup.v4
run/setup.v4: gro-off enable-ipv4 tc-on ## Run setup in Virtual IPv4 mode
	@echo "Virtual IPv4 mode setup completed"

.PHONY: run/setup.v6
run/setup.v6: gro-off enable-ipv6 ## Run setup in Virtual IPv6 mode
	@echo "Virtual IPv6 mode setup completed"


# builds
#===============================================================
build:
	GOOS=linux $(GOBUILD) ./cmd/adapterd/adapterd.go

build/pi:
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) ./cmd/adapterd/adapterd.go

build/pi64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) ./cmd/adapterd/adapterd.go



# develop
#===============================================================
.PHONY: scp ## Transfer this directory
scp:
	@tar zcf - -C ../ --format=ustar adapterd | ssh ${ADAPTER_HOSTUSER}@${ADAPTER_ENDPOINT} "tar zxvf - -C /home/${ADAPTER_HOSTUSER}/src/develop"

.PHONY: ch_route
ch_route: ## Change the default route
	@sudo ./scripts/networking/change-default-route.sh

.PHONY: gro-off
gro-off: ## Do not use GRO
	@sudo ./scripts/networking/generic-receive-offload.sh

.PHONY: tso-off
tso-off: ## Do not use TSO
	@sudo ./scripts/networking/tcp-segmentation-offload.sh

.PHONY: enable-ipv4
enable-ipv4: ## Enable IPv4
	@sudo ./scripts/networking/enable-ipv4.sh

.PHONY: enable-ipv6
enable-ipv6: ## Enable IPv6
	@sudo ./scripts/networking/enable-ipv6.sh

.PHONY: tc-on
tc-on: ## Turn on traffic control
	@sudo ./scripts/ebpf/traffic-control.sh


# tests
#===============================================================
.PHONY: lint
lint: ## golang-ci lint
	$(GOLANGCI) run --config=.golangci.yml ./...

.PHONY: gtest
gtest: ## go test
	$(GOTEST) -v ./...

.PHONY: benchmark
benchmark: ## go benchmark
	$(GOTEST) -bench -v ./...

.PHONY: doc
doc: ## godoc http:6060
	$(GODOC) -http=:6060

.PHONY: coverage
coverage: ## coverage
	go test -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o cover.html



# Makefile config
#===============================================================
help:
	echo "Usage: make [task]\n\nTasks:"
	perl -nle 'printf("    \033[33m%-30s\033[0m %s\n",$$1,$$2) if /^([a-zA-Z0-9_\/-]*?):(?:.+?## )?(.*?)$$/' $(MAKEFILE_LIST)

.SILENT: help

.PHONY: $(shell egrep -o '^(\._)?[a-z_-]+:' $(MAKEFILE_LIST) | sed 's/://')
