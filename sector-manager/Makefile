SHELL := /bin/bash

IMAGE_NAME = sector-manager

NODE_ID ?= node1
BOOTSTRAP ?= false
PEERS ?=

# Configurações de rede
NETWORK_MODE ?= bridge
RAFT_HOST ?= 127.0.0.1
RAFT_PORT ?= 10001
SIG_HOST ?= 127.0.0.1
SIG_PORT ?= 9123
BROKER_HOST ?= 127.0.0.1
BROKER_PORT ?= 1883

DATA_DIR ?= $(shell pwd)/data/$(NODE_ID)
RUN_ARGS ?=

# Captura UID e GID para evitar problemas de permissão na pasta data
USER_ID = $(shell id -u)
GROUP_ID = $(shell id -g)

.PHONY: build run-node run-node-prompt clean help prepare

help:
	@echo "Available targets:"
	@echo "  build              - Builda a imagem docker"
	@echo "  run-node           - Sobe um node (NETWORK_MODE=bridge ou host)"
	@echo "  run-node-prompt    - Sobe um nó com configuração interativa"
	@echo "  clean              - Remove contêineres e apaga pasta de data"

build:
	@echo "Compilando imagem Docker..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

# Garante que a pasta de dados existe antes do Docker rodar para manter o dono correto
prepare:
	mkdir -p $(DATA_DIR)

run-node: prepare
	@echo "Iniciando nó $(NODE_ID) em modo $(NETWORK_MODE)..."
	@if [ "$(NETWORK_MODE)" = "host" ]; then \
		docker run -it --rm \
			--name $(NODE_ID) \
			--network host \
			--user $(USER_ID):$(GROUP_ID) \
			-v $(DATA_DIR):/data \
			$(RUN_ARGS) \
			$(IMAGE_NAME) \
			-id=$(NODE_ID) \
			-raft=$(RAFT_HOST):$(RAFT_PORT) \
			-dir=/data \
			-sig=$(SIG_HOST):$(SIG_PORT) \
			-broker=tcp://$(BROKER_HOST):$(BROKER_PORT) \
			$(if $(PEERS),-peers=$(PEERS),) \
			$(if $(filter true,$(BOOTSTRAP)),-bootstrap=true,); \
	else \
		docker run -it --rm \
			--name $(NODE_ID) \
			-p $(RAFT_PORT):$(RAFT_PORT) \
			-p $(SIG_PORT):$(SIG_PORT) \
			--user $(USER_ID):$(GROUP_ID) \
			-v $(DATA_DIR):/data \
			$(RUN_ARGS) \
			$(IMAGE_NAME) \
			-id=$(NODE_ID) \
			-raft=$(RAFT_HOST):$(RAFT_PORT) \
			-dir=/data \
			-sig=$(SIG_HOST):$(SIG_PORT) \
			-broker=tcp://$(BROKER_HOST):$(BROKER_PORT) \
			$(if $(PEERS),-peers=$(PEERS),) \
			$(if $(filter true,$(BOOTSTRAP)),-bootstrap=true,); \
	fi

run-node-prompt:
	@echo "Configuracao interativa do sector-manager"
	@read -p "NODE_ID [$(NODE_ID)]: " _NODE_ID; \
	_NODE_ID=$${_NODE_ID:-$(NODE_ID)}; \
	read -p "NETWORK_MODE (bridge/host) [$(NETWORK_MODE)]: " _NET; \
	_NET=$${_NET:-$(NETWORK_MODE)}; \
	read -p "BOOTSTRAP (true/false) [$(BOOTSTRAP)]: " _BOOT; \
	_BOOT=$${_BOOT:-$(BOOTSTRAP)}; \
	read -p "PEERS (ip1:port,ip2:port) [$(PEERS)]: " _PEERS; \
	_PEERS=$${_PEERS:-$(PEERS)}; \
	read -p "RAFT_HOST [$(RAFT_HOST)] (somente IP): " _R_HOST; \
	_R_HOST=$${_R_HOST:-$(RAFT_HOST)}; \
	read -p "RAFT_PORT [$(RAFT_PORT)] (somente PORTA): " _R_PORT; \
	_R_PORT=$${_R_PORT:-$(RAFT_PORT)}; \
	read -p "SIG_HOST [$(SIG_HOST)] (somente IP): " _S_HOST; \
	_S_HOST=$${_S_HOST:-$(SIG_HOST)}; \
	read -p "SIG_PORT [$(SIG_PORT)] (somente PORTA): " _S_PORT; \
	_S_PORT=$${_S_PORT:-$(SIG_PORT)}; \
	read -p "BROKER_HOST [$(BROKER_HOST)] (somente IP): " _B_HOST; \
	_B_HOST=$${_B_HOST:-$(BROKER_HOST)}; \
	read -p "BROKER_PORT [$(BROKER_PORT)] (somente PORTA): " _B_PORT; \
	_B_PORT=$${_B_PORT:-$(BROKER_PORT)}; \
	read -p "DATA_DIR [$(DATA_DIR)]: " _DIR; \
	_DIR=$${_DIR:-$(DATA_DIR)}; \
	$(MAKE) run-node \
		NODE_ID=$$_NODE_ID NETWORK_MODE=$$_NET BOOTSTRAP=$$_BOOT PEERS="$$_PEERS" \
		RAFT_HOST=$$_R_HOST RAFT_PORT=$$_R_PORT \
		SIG_HOST=$$_S_HOST SIG_PORT=$$_S_PORT \
		BROKER_HOST=$$_B_HOST BROKER_PORT=$$_B_PORT \
		DATA_DIR=$$_DIR

clean:
	@echo "Limpando containers e dados..."
	-docker stop $(NODE_ID)
	-docker rm $(NODE_ID)
	rm -rf data/