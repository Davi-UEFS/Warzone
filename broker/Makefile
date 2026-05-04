SHELL := /bin/bash

IMAGE_NAME = sector-broker
BROKER_NAME ?= broker1
BROKER_PORT ?= 1883
# Variável para alternar entre rede Bridge (padrão) e Host (real)
NETWORK_MODE ?= bridge 

.PHONY: build run-broker run-broker-prompt clean help

help:
	@echo "Available targets:"
	@echo "  build              - Builda a imagem docker"
	@echo "  run-broker         - Roda o broker (use NETWORK_MODE=host para rede real)"
	@echo "  run-broker-prompt  - Sobe um nó com configuração interativa"
	@echo "  clean              - Para e remove o broker"

build:
	@echo "Compilando imagem Docker..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

run-broker:
	@echo "Iniciando broker $(BROKER_NAME) em modo $(NETWORK_MODE)..."
	@if [ "$(NETWORK_MODE)" = "host" ]; then \
		docker run -it --rm --name $(BROKER_NAME) --network host $(IMAGE_NAME); \
	else \
		docker run -it --rm --name $(BROKER_NAME) -p $(BROKER_PORT):1883 $(IMAGE_NAME); \
	fi

run-broker-prompt:
	@echo "Configuracao interativa do broker"
	@read -p "BROKER_NAME [$(BROKER_NAME)]: " _name; \
	_name=$${_name:-$(BROKER_NAME)}; \
	read -p "BROKER_PORT [$(BROKER_PORT)]: " _port; \
	_port=$${_port:-$(BROKER_PORT)}; \
	read -p "NETWORK_MODE (bridge/host) [$(NETWORK_MODE)]: " _net; \
	_net=$${_net:-$(NETWORK_MODE)}; \
	$(MAKE) run-broker BROKER_NAME=$$_name BROKER_PORT=$$_port NETWORK_MODE=$$_net

clean:
	@echo "Limpando container..."
	-docker stop $(BROKER_NAME)
	-docker rm $(BROKER_NAME)