SHELL := /bin/bash

IMAGE_NAME = sector-broker
BROKER_NAME ?= broker1
BROKER_PORT ?= 1883
BRIDGE_NETWORK ?= warzone-net

.PHONY: build run-broker-host run-broker-bridge clean help

help:
	@echo "Available targets:"
	@echo "  build              - Builda a imagem docker"
	@echo "  run-broker-host    - Sobe o broker em modo host"
	@echo "  run-broker-bridge  - Sobe o broker em modo bridge"
	@echo "  clean              - Para e remove o broker"
	@echo ""
	@echo "Examples:"
	@echo "  make run-broker-host"
	@echo "  make run-broker-bridge BRIDGE_NETWORK=sector1-net"

build:
	@echo "Compilando imagem Docker..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

run-broker-host:
	@echo "Configuracao interativa do broker (host)"; \
	read -p "BROKER_NAME [$(BROKER_NAME)]: " _name; \
	_name=$${_name:-$(BROKER_NAME)}; \
	read -p "BROKER_PORT [$(BROKER_PORT)]: " _port; \
	_port=$${_port:-$(BROKER_PORT)}; \
	echo "Iniciando broker $$_name em modo host..."; \
	docker run -it --rm -p $$_port:$$_port --name $$_name $(IMAGE_NAME) -id=$$_name -port=$$_port


clean:
	@echo "Limpando container..."
	-docker stop $(BROKER_NAME)
	-docker rm $(BROKER_NAME)