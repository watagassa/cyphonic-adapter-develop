GOCMD=go
GORUN=$(GOCMD) run
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GODOC=$(GOCMD)doc
DOCKER=docker
COMPOSE=$(DOCKER) compose
EXEC=$(COMPOSE) exec
EXECD=$(COMPOSE) exec -d
BUILD=$(COMPOSE) build
UP=$(COMPOSE) up -d
LOGS=$(COMPOSE) logs
STOP=$(COMPOSE) stop
RM=$(COMPOSE) rm
DOWN=$(COMPOSE) down
ADAPTER=$(EXEC) adapter
SHELL=sh



# develop for docker-compose
#===============================================================
.PHONY: init
init: create-external-network up route-priority

.PHONY: build
build: ## docker build
	${BUILD}

.PHONY: up
up: ## docker up
	${UP}

.PHONY: logs
logs: ${LOGS} ## docker logs

.PHONY: adapter/bash
adapter/bash: ## exec adapter container
	${ADAPTER} bash

.PHONY: gn01/bash
gn01/bash: ## exec general-node01 container
	${EXEC} general-node01 bash

.PHONY: logs/adapter
logs/adapter: ## logs adapter container
	${LOGS} adapter

.PHONY: logs/gn01
logs/gn01: ## logs general-node01 container
	${LOGS} general-node01

.PHONY: stop
stop: ## docker stop
	${COMPOSE} stop

.PHONY: down
down: ## docker down
	${COMPOSE} down

.PHONY: down/all
down/all: ## delete images, network, containers
	${DOCKER} system prune --all

.PHONY: down/vol
down/vol: ## delete volumes
	${DOCKER} volume prune



# Script
#===============================================================
.PHONY: create-external-network
create-external-network: ## create an external network
	${SHELL} .docker/adapter/scripts/external-network.sh

.PHONY: route-priority
route-priority: ## prefer IPv4
	${ADAPTER} ${SHELL} /go/src/github.com/Pluslab/cyphonic-adapter/.docker/adapter/scripts/route-priority.sh



# Environment
#===============================================================
.PHONY: plugin-install
plugin-install: ## Install asdf plugins
	@${SHELL} ./bin/scripts/plugin.sh



# tests
#===============================================================
.PHONY: doc
doc: ## godoc http:6060
	${GODOC} -http=:6060



# Makefile config
#===============================================================
help: ## Display this help screen
	@grep -E '^[a-zA-Z/_-]+:.*?## .*$$' ${MAKEFILE_LIST} | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
