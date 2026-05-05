SHELL := /bin/bash

IMAGE_NAME = sector-manager

NODE_ID ?= node1
HOST ?= 127.0.0.1
RAFT_PORT ?= 10001
SIG_PORT ?= 9123
BROKER_HOST ?= 127.0.0.1
BROKER_PORT ?= 1883
BOOTSTRAP ?= false
PEERS ?=
DATA_DIR ?= $(shell pwd)/dataRaft/$(NODE_ID)
RUN_ARGS ?=

USER_ID = $(shell id -u)
GROUP_ID = $(shell id -g)

.PHONY: build run-node-host clean help

help:
	@echo "Targets:"
	@echo "  build         - Builda a imagem"
	@echo "  run-node-host - Sobe um node em modo host"
	@echo "  clean         - Limpa containers e dados"
	@echo ""
	@echo "Exemplo:"
	@echo "  make run-node-host NODE_ID=node1 BOOTSTRAP=true"
	@echo "  make run-node-host NODE_ID=node2 PEERS=192.168.0.10:9123 RAFT_PORT=10002 SIG_PORT=9124"

build:
	@echo "Compilando imagem Docker..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

run-node-host:
	@echo "Configuracao interativa (host)"
	@echo "Pressione ENTER para usar o padrao"
	@echo ""
	@echo "O id do nó"; \
	echo "NODE_ID [padrao: $(NODE_ID)]:"; \
	read _NODE_ID; \
	_NODE_ID=$${_NODE_ID:-$(NODE_ID)}; \
	echo "Se este nó vai iniciar como lider"; \
	echo "BOOTSTRAP (true/false) [padrao: $(BOOTSTRAP)]:"; \
	read _BOOT; \
	_BOOT=$${_BOOT:-$(BOOTSTRAP)}; \
	echo "Lista de peers (SIG host:porta), separados por virgula"; \
	echo "PEERS [padrao: $(PEERS)]:"; \
	read _PEERS; \
	_PEERS=$${_PEERS:-$(PEERS)}; \
	echo "Host base para Raft e SIG (modo host)"; \
	echo "HOST [padrao: $(HOST)]:"; \
	read _HOST; \
	_HOST=$${_HOST:-$(HOST)}; \
	echo "Porta do Raft"; \
	echo "RAFT_PORT [padrao: $(RAFT_PORT)]:"; \
	read _RAFT_PORT; \
	_RAFT_PORT=$${_RAFT_PORT:-$(RAFT_PORT)}; \
	echo "Porta do servidor de sinalizacao"; \
	echo "SIG_PORT [padrao: $(SIG_PORT)]:"; \
	read _SIG_PORT; \
	_SIG_PORT=$${_SIG_PORT:-$(SIG_PORT)}; \
	echo "Host do broker MQTT"; \
	echo "BROKER_HOST [padrao: $(BROKER_HOST)]:"; \
	read _BROKER_HOST; \
	_BROKER_HOST=$${_BROKER_HOST:-$(BROKER_HOST)}; \
	echo "Porta do broker MQTT"; \
	echo "BROKER_PORT [padrao: $(BROKER_PORT)]:"; \
	read _BROKER_PORT; \
	_BROKER_PORT=$${_BROKER_PORT:-$(BROKER_PORT)}; \
	_DIR="$(CURDIR)/dataRaft/$$_NODE_ID"; \
	mkdir -p "$$_DIR"; \
	PEERS_ARG=""; \
	if [ -n "$$_PEERS" ]; then PEERS_ARG="-peers=$$_PEERS"; fi; \
	BOOT_ARG=""; \
	if [ "$$_BOOT" = "true" ]; then BOOT_ARG="-bootstrap=true"; fi; \
	echo "Iniciando no $$_NODE_ID em modo host..."; \
	docker run -it --rm \
		--name $$_NODE_ID \
		--network host \
		--user $(USER_ID):$(GROUP_ID) \
		-v "$$_DIR":/data \
		$(RUN_ARGS) \
		$(IMAGE_NAME) \
		-id=$$_NODE_ID \
		-raft=$$_HOST:$$_RAFT_PORT \
		-sig=$$_HOST:$$_SIG_PORT \
		-broker=tcp://$$_BROKER_HOST:$$_BROKER_PORT \
		-dir=/data \
		$$PEERS_ARG \
		$$BOOT_ARG

clean:
	@echo "Limpando containers e dados..."
	-docker stop $(NODE_ID) 2>/dev/null
	-docker rm $(NODE_ID) 2>/dev/null
	rm -rf dataRaft/