SHELL := /bin/bash

IMAGE_NAME = sector-manager

# Detecta o IP da rede local automaticamente
# Tenta primeiro via hostname -I, se falhar usa ip route
DETECTED_IP := $(shell hostname -I | awk '{print $$1}')
ifeq ($(DETECTED_IP),)
    DETECTED_IP := $(shell ip route get 1 | awk '{print $$7;exit}')
endif

NODE_ID ?= node1
BOOTSTRAP ?= false
PEERS ?=

# Configurações de rede
NETWORK_MODE ?= host
LOCAL_HOST ?= $(DETECTED_IP)
RAFT_PORT ?= 10001
SIG_PORT ?= 9123
BROKER_PORT ?= 1883

DATA_DIR ?= $(shell pwd)/dataRaft/$(NODE_ID)
RUN_ARGS ?=

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
			-raft=$(LOCAL_HOST):$(RAFT_PORT) \
			-dir=/data \
			-sig=$(LOCAL_HOST):$(SIG_PORT) \
			-broker=tcp://$(LOCAL_HOST):$(BROKER_PORT) \
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
			-raft=$(LOCAL_HOST):$(RAFT_PORT) \
			-dir=/data \
			-sig=$(LOCAL_HOST):$(SIG_PORT) \
			-broker=tcp://$(LOCAL_HOST):$(BROKER_PORT) \
			$(if $(PEERS),-peers=$(PEERS),) \
			$(if $(filter true,$(BOOTSTRAP)),-bootstrap=true,); \
	fi

run-node-prompt:
	@echo "Configuracao interativa do sector-manager"; \
	echo "---"; \
	echo "Identificador do nó (ex: node1, node2)"; \
	read -p "NODE_ID [Padrão: $(NODE_ID)]: " _NODE_ID; \
	_NODE_ID=$${_NODE_ID:-$(NODE_ID)}; \
	echo "---"; \
	echo "Modo de rede (host para rede física, bridge para isolada)"; \
	read -p "NETWORK_MODE [Padrão: $(NETWORK_MODE)]: " _NET; \
	_NET=$${_NET:-$(NETWORK_MODE)}; \
	echo "---"; \
	echo "Este nó será o líder inicial?"; \
	read -p "BOOTSTRAP (true/false) [Padrão: $(BOOTSTRAP)]: " _BOOT; \
	_BOOT=$${_BOOT:-$(BOOTSTRAP)}; \
	echo "---"; \
	echo "Lista de outros nós (ex: 192.168.1.10:9123,192.168.1.11:9123)"; \
	read -p "PEERS [Padrão: nenhum]: " _PEERS; \
	_PEERS=$${_PEERS:-$(PEERS)}; \
	echo "---"; \
	echo "IP da máquina local (insira apenas se não quiser usar o hostname detectado automaticamente)"; \
	read -p "LOCAL_HOST [Padrão: $(LOCAL_HOST)]: " _L_HOST; \
	_L_HOST=$${_L_HOST:-$(LOCAL_HOST)}; \
	echo "---"; \
	echo "Porta para o algoritmo Raft"; \
	read -p "RAFT_PORT [Padrão: $(RAFT_PORT)]: " _R_PORT; \
	_R_PORT=$${_R_PORT:-$(RAFT_PORT)}; \
	echo "---"; \
	echo "Porta para escuta de outros nós (SIG)"; \
	read -p "SIG_PORT [Padrão: $(SIG_PORT)]: " _S_PORT; \
	_S_PORT=$${_S_PORT:-$(SIG_PORT)}; \
	echo "---"; \
	echo "Porta do broker MQTT"; \
	read -p "BROKER_PORT [Padrão: $(BROKER_PORT)]: " _B_PORT; \
	_B_PORT=$${_B_PORT:-$(BROKER_PORT)}; \
	echo "---"; \
	echo "Caminho da pasta de dados. Pode usar caminho relativo"; \
	read -p "DATA_DIR [Padrão: $(DATA_DIR)]: " _DIR; \
	_DIR=$${_DIR:-$(DATA_DIR)}; \
	$(MAKE) run-node \
		NODE_ID=$$_NODE_ID NETWORK_MODE=$$_NET BOOTSTRAP=$$_BOOT PEERS="$$_PEERS" \
		LOCAL_HOST=$$_L_HOST RAFT_PORT=$$_R_PORT SIG_PORT=$$_S_PORT BROKER_PORT=$$_B_PORT \
		DATA_DIR=$$_DIR

clean:
	@echo "Limpando containers e dados..."
	-docker stop $(NODE_ID) 2>/dev/null
	-docker rm $(NODE_ID) 2>/dev/null
	rm -rf dataRaft/