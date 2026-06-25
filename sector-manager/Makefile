SHELL := /bin/bash

IMAGE_NAME = daviuefs/sector-manager
NODE1_ID ?= setorA
NODE2_ID ?= setorB
NODE3_ID ?= setorC

NODE1_DASHBOARD_PORT ?= 8081
NODE2_DASHBOARD_PORT ?= 8082
NODE3_DASHBOARD_PORT ?= 8083

BROKER1_PORT ?= 1883
BROKER2_PORT ?= 1884
BROKER3_PORT ?= 1885

USER_ID = $(shell id -u)
GROUP_ID = $(shell id -g)
DATA_BASE_DIR ?= $(shell pwd)/dataRaft

FLAGS ?=

.PHONY: build run-all run-sectorA run-sectorA-go run-sectorB run-sectorB-go run-sectorC run-sectorC-go run-node-prompt clean help

help:
	@echo "Targets:"
	@echo "  build           - Builda a imagem dos nodes (Manager + Broker Embutido)"
	@echo "  run-sectorA     - Sobe o nó A (Líder inicial) via Docker"
	@echo "  run-sectorA-go  - Sobe o nó A (Líder inicial) via Go nativo"
	@echo "  run-sectorB     - Sobe o nó B"
	@echo "  run-sectorC     - Sobe o nó C"
	@echo "  run-node-prompt - Sobe 1 node com configuracao interativa"
	@echo "  clean           - Remove containers e dados"
	@echo ""
	@echo "DICA: Abra 3 terminais e rode run-sectorA, run-sectorB e run-sectorC separadamente."
	@echo "USO DE FLAGS: make run-sectorA-go FLAGS=\"-debug\""

build:
	@echo "Compilando imagem Docker dos nodes..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

pull:
	@echo "Baixando imagem do Docker Hub..."
	docker pull $(IMAGE_NAME):latest

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
		-broker-port=$(BROKER1_PORT) \
		-dashboard-port=$(NODE1_DASHBOARD_PORT) \
		$(FLAGS)

run-sectorA-go:
	@mkdir -p $(DATA_BASE_DIR)/$(NODE1_ID)
	@chmod -R 777 $(DATA_BASE_DIR)/$(NODE1_ID)
	go run . \
		-id=$(NODE1_ID) \
		-broker-port=$(BROKER1_PORT) \
		-dashboard-port=$(NODE1_DASHBOARD_PORT) \
		$(FLAGS)

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
		-broker-port=$(BROKER2_PORT) \
		-dashboard-port=$(NODE2_DASHBOARD_PORT) \
		$(FLAGS)

run-sectorB-go:
	@mkdir -p $(DATA_BASE_DIR)/$(NODE2_ID)
	@chmod -R 777 $(DATA_BASE_DIR)/$(NODE2_ID)
	go run . \
		-id=$(NODE2_ID) \
		-broker-port=$(BROKER2_PORT) \
		-dashboard-port=$(NODE2_DASHBOARD_PORT) \
		$(FLAGS)

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
		-broker-port=$(BROKER3_PORT) \
		-dashboard-port=$(NODE3_DASHBOARD_PORT) \
		$(FLAGS)

run-sectorC-go:
	@mkdir -p $(DATA_BASE_DIR)/$(NODE3_ID)
	@chmod -R 777 $(DATA_BASE_DIR)/$(NODE3_ID)
	go run . \
		-id=$(NODE3_ID) \
		-broker-port=$(BROKER3_PORT) \
		-dashboard-port=$(NODE3_DASHBOARD_PORT) \
		$(FLAGS)

run-node-prompt:
	@echo "Configuracao interativa (host)"
	@echo "Pressione ENTER para usar o padrao"
	@echo ""
	@echo "O id do nó"; \
	echo "NODE_ID [padrao: $(NODE1_ID)]:"; \
	read _NODE_ID; \
	_NODE_ID=$${_NODE_ID:-$(NODE1_ID)}; \
	echo "Porta do broker MQTT embutido"; \
	echo "BROKER_PORT [padrao: $(BROKER1_PORT)]:"; \
	read _BROKER_PORT; \
	_BROKER_PORT=$${_BROKER_PORT:-$(BROKER1_PORT)}; \
	echo "Porta do dashboard HTTP"; \
	echo "DASHBOARD_PORT [padrao: $(NODE1_DASHBOARD_PORT)]:"; \
	read _DASHBOARD_PORT; \
	_DASHBOARD_PORT=$${_DASHBOARD_PORT:-$(NODE1_DASHBOARD_PORT)}; \
	echo "Iniciando no $$_NODE_ID em modo host..."; \
	mkdir -p "$(CURDIR)/dataRaft/$$_NODE_ID"; \
	chmod -R 777 "$(CURDIR)/dataRaft/$$_NODE_ID"; \
	docker run -it --rm \
		--name $$_NODE_ID \
		--network host \
		--user $(USER_ID):$(GROUP_ID) \
		-v "$(CURDIR)/dataRaft/$$_NODE_ID":/data \
		$(IMAGE_NAME) \
		-id=$$_NODE_ID \
		-broker-port=$$_BROKER_PORT \
		-dashboard-port=$$_DASHBOARD_PORT \
		$(FLAGS)

clean:
	@echo "Limpando containers e dados dos nodes..."
	-@docker stop $(NODE1_ID) $(NODE2_ID) $(NODE3_ID) 2>/dev/null || true
	-@docker rm $(NODE1_ID) $(NODE2_ID) $(NODE3_ID) 2>/dev/null || true
	rm -rf $(DATA_BASE_DIR)

KEYRING_BASE_DIR ?= $(HOME)/warzone-data
BLOCKCHAIN_REST_URLS ?= http://localhost:1317,http://localhost:1318,http://localhost:1319,http://localhost:1320
BLOCKCHAIN_RPC_URL ?= http://localhost:26657
CHAIN_ID ?= warzone-rede

run-sectorA-blockchain:
	docker run -d --name manager-a \
		--network host \
		--pull never \
		-v $(KEYRING_BASE_DIR)/setorA:/keyring \
		-e SECTOR_ID=Setor-A \
		-e WALLET_NAME=key_a \
		-e KEYRING_DIR=/keyring \
		-e BLOCKCHAIN_REST_URLS="$(BLOCKCHAIN_REST_URLS)" \
		-e BLOCKCHAIN_RPC_URL="$(BLOCKCHAIN_RPC_URL)" \
		-e CHAIN_ID="$(CHAIN_ID)" \
		$(IMAGE_NAME) --id setor-a --broker-port=1884
	docker logs -f manager-a

run-sectorB-blockchain:
	docker run -d --name manager-b \
		--network host \
		--pull never \
		-v $(KEYRING_BASE_DIR)/setorB:/keyring \
		-e SECTOR_ID=Setor-B \
		-e WALLET_NAME=key_b \
		-e KEYRING_DIR=/keyring \
		-e BLOCKCHAIN_REST_URLS="$(BLOCKCHAIN_REST_URLS)" \
		-e BLOCKCHAIN_RPC_URL="$(BLOCKCHAIN_RPC_URL)" \
		-e CHAIN_ID="$(CHAIN_ID)" \
		$(IMAGE_NAME) --id setor-b --broker-port=1884
	docker logs -f manager-b

run-sectorC-blockchain:
	docker run -d --name manager-c \
		--network host \
		--pull never \
		-v $(KEYRING_BASE_DIR)/setorC:/keyring \
		-e SECTOR_ID=Setor-C \
		-e WALLET_NAME=key_c \
		-e KEYRING_DIR=/keyring \
		-e BLOCKCHAIN_REST_URLS="$(BLOCKCHAIN_REST_URLS)" \
		-e BLOCKCHAIN_RPC_URL="$(BLOCKCHAIN_RPC_URL)" \
		-e CHAIN_ID="$(CHAIN_ID)" \
		$(IMAGE_NAME) --id setor-c --broker-port=1884
	docker logs -f manager-c

run-sectorD-blockchain:
	docker run -d --name manager-d \
		--network host \
		--pull never \
		-v $(KEYRING_BASE_DIR)/setorD:/keyring \
		-e SECTOR_ID=Setor-D \
		-e WALLET_NAME=key_d \
		-e KEYRING_DIR=/keyring \
		-e BLOCKCHAIN_REST_URLS="$(BLOCKCHAIN_REST_URLS)" \
		-e BLOCKCHAIN_RPC_URL="$(BLOCKCHAIN_RPC_URL)" \
		-e CHAIN_ID="$(CHAIN_ID)" \
		$(IMAGE_NAME) --id setor-d
	docker logs -f manager-d

stop-blockchain:
	-docker stop manager-a manager-b manager-c manager-d 2>/dev/null || true
	-docker rm manager-a manager-b manager-c manager-d 2>/dev/null || true

# ==========================================
# DASHBOARD TÁTICO E SECTOR MANAGER
# ==========================================
SECTOR_ID ?= Setor-A
BLOCKCHAIN_URL ?= http://localhost:1317
BROKER_URL ?= tcp://localhost:1883
PORT ?= 8080

.PHONY: run-dashboard

run-dashboard:
	@echo "\033[1;34m======================================================\033[0m"
	@echo "\033[1;32m🚀 INICIANDO SECTOR MANAGER E DASHBOARD TÁTICO\033[0m"
	@echo "📡 Setor Alvo: $(SECTOR_ID)"
	@echo "🔗 Blockchain: $(BLOCKCHAIN_URL)"
	@echo "🌐 Painel Web: http://localhost:$(PORT)"
	@echo "\033[1;34m======================================================\033[0m"
	SECTOR_ID="$(SECTOR_ID)" \
	BLOCKCHAIN_REST_URLS="$(BLOCKCHAIN_URL)" \
	go run . -broker="$(BROKER_URL)" -port=$(PORT)