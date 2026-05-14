SHELL := /bin/bash

IMAGE_NAME = sector-manager
HOST ?= 127.0.0.1
NODE1_ID ?= setorA
NODE2_ID ?= setorB
NODE3_ID ?= setorC

NODE1_RAFT_PORT ?= 10001
NODE2_RAFT_PORT ?= 10002
NODE3_RAFT_PORT ?= 10003

NODE1_SIG_PORT ?= 11001
NODE2_SIG_PORT ?= 11002
NODE3_SIG_PORT ?= 11003

# Agora o Broker roda embutido no manager de cada setor
BROKER1_PORT ?= 1883
BROKER2_PORT ?= 1884
BROKER3_PORT ?= 1885

USER_ID = $(shell id -u)
GROUP_ID = $(shell id -g)
DATA_BASE_DIR ?= $(shell pwd)/dataRaft

.PHONY: build run-all run-sectorA run-sectorB run-sectorC run-node-prompt clean help

help:
	@echo "Targets:"
	@echo "  build           - Builda a imagem dos nodes (Manager + Broker Embutido)"
	@echo "  run-sectorA     - Sobe o nó A (Líder inicial)"
	@echo "  run-sectorB     - Sobe o nó B"
	@echo "  run-sectorC     - Sobe o nó C"
	@echo "  run-node-prompt - Sobe 1 node com configuracao interativa"
	@echo "  clean           - Remove containers e dados"
	@echo ""
	@echo "DICA: Abra 3 terminais e rode run-sectorA, run-sectorB e run-sectorC separadamente."

build:
	@echo "Compilando imagem Docker dos nodes..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

run-all: 
	@echo "Para ver os logs corretamente, execute run-sectorA, run-sectorB e run-sectorC em terminais separados!"

run-sectorA:
	@mkdir -p $(DATA_BASE_DIR)/$(NODE1_ID)
	@chmod -R 777 $(DATA_BASE_DIR)/$(NODE1_ID)
	docker run -it --rm \
		--name $(NODE1_ID) \
		--network host \
		--user $(USER_ID):$(GROUP_ID) \
		-v "$(DATA_BASE_DIR)/$(NODE1_ID)":/data \
		$(IMAGE_NAME) \
		-id=$(NODE1_ID) \
		-host=$(HOST) \
		-raft-port=$(NODE1_RAFT_PORT) \
		-broker-port=$(BROKER1_PORT) \
		-dir=/data \
		-peers=$(HOST):$(NODE2_SIG_PORT),$(HOST):$(NODE3_SIG_PORT) \
		-bootstrap=true

run-sectorB:
	@mkdir -p $(DATA_BASE_DIR)/$(NODE2_ID)
	@chmod -R 777 $(DATA_BASE_DIR)/$(NODE2_ID)
	docker run -it --rm \
		--name $(NODE2_ID) \
		--network host \
		--user $(USER_ID):$(GROUP_ID) \
		-v "$(DATA_BASE_DIR)/$(NODE2_ID)":/data \
		$(IMAGE_NAME) \
		-id=$(NODE2_ID) \
		-host=$(HOST) \
		-raft-port=$(NODE2_RAFT_PORT) \
		-broker-port=$(BROKER2_PORT) \
		-dir=/data \
		-peers=$(HOST):$(NODE1_SIG_PORT),$(HOST):$(NODE3_SIG_PORT)

run-sectorC:
	@mkdir -p $(DATA_BASE_DIR)/$(NODE3_ID)
	@chmod -R 777 $(DATA_BASE_DIR)/$(NODE3_ID)
	docker run -it --rm \
		--name $(NODE3_ID) \
		--network host \
		--user $(USER_ID):$(GROUP_ID) \
		-v "$(DATA_BASE_DIR)/$(NODE3_ID)":/data \
		$(IMAGE_NAME) \
		-id=$(NODE3_ID) \
		-host=$(HOST) \
		-raft-port=$(NODE3_RAFT_PORT) \
		-broker-port=$(BROKER3_PORT) \
		-dir=/data \
		-peers=$(HOST):$(NODE1_SIG_PORT),$(HOST):$(NODE2_SIG_PORT)

run-node-prompt:
	@echo "Configuracao interativa (host)"
	@echo "Pressione ENTER para usar o padrao"
	@echo ""
	@echo "O id do nó"; \
	echo "NODE_ID [padrao: $(NODE1_ID)]:"; \
	read _NODE_ID; \
	_NODE_ID=$${_NODE_ID:-$(NODE1_ID)}; \
	echo "Se este nó vai iniciar como lider"; \
	echo "BOOTSTRAP (true/false) [padrao: false]:"; \
	read _BOOT; \
	_BOOT=$${_BOOT:-false}; \
	echo "Lista de peers (SIG host:porta), separados por virgula"; \
	echo "PEERS [padrao: ]:"; \
	read _PEERS; \
	_PEERS=$${_PEERS:-}; \
	echo "Host base para Raft e SIG (modo host)"; \
	echo "HOST [padrao: $(HOST)]:"; \
	read _HOST; \
	_HOST=$${_HOST:-$(HOST)}; \
	echo "Porta do Raft"; \
	echo "RAFT_PORT [padrao: 10001]:"; \
	read _RAFT_PORT; \
	_RAFT_PORT=$${_RAFT_PORT:-10001}; \
	echo "Porta do broker MQTT embutido"; \
	echo "BROKER_PORT [padrao: $(BROKER1_PORT)]:"; \
	read _BROKER_PORT; \
	_BROKER_PORT=$${_BROKER_PORT:-$(BROKER1_PORT)}; \
	_DIR="$(CURDIR)/dataRaft/$$_NODE_ID"; \
	mkdir -p "$$_DIR"; \
	PEERS_ARG=""; \
	if [ -n "$$_PEERS" ]; then PEERS_ARG="-peers=$$_PEERS"; fi; \
	BOOT_ARG=""; \
	if [ "$$_BOOT" = "true" ]; then BOOT_ARG="-bootstrap=true"; fi; \
	echo "Iniciando no $$_NODE_ID em modo host..."; \
	mkdir -p "$$_DIR"; \
	chmod -R 777 "$$_DIR"; \
	docker run -it --rm \
		--name $$_NODE_ID \
		--network host \
		--user $(USER_ID):$(GROUP_ID) \
		-v "$$_DIR":/data \
		$(IMAGE_NAME) \
		-id=$$_NODE_ID \
		-host=$$_HOST \
		-raft-port=$$_RAFT_PORT \
		-broker-port=$$_BROKER_PORT \
		-dir=/data \
		$$PEERS_ARG \
		$$BOOT_ARG

clean:
	@echo "Limpando containers e dados dos nodes..."
	-@docker stop $(NODE1_ID) $(NODE2_ID) $(NODE3_ID) 2>/dev/null || true
	-@docker rm $(NODE1_ID) $(NODE2_ID) $(NODE3_ID) 2>/dev/null || true
	rm -rf $(DATA_BASE_DIR)