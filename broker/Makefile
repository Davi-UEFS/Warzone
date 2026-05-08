SHELL := /bin/bash

IMAGE_NAME = sector-broker

BROKER1_NAME ?= broker1
BROKER2_NAME ?= broker2
BROKER3_NAME ?= broker3

BROKER1_PORT ?= 1883
BROKER2_PORT ?= 1884
BROKER3_PORT ?= 1885

.PHONY: build run-all run-broker1 run-broker2 run-broker3 run-broker-prompt clean help

help:
	@echo "Targets:"
	@echo "  build             - Builda a imagem dos brokers"
	@echo "  run-all           - Sobe os 3 brokers"
	@echo "  run-broker-prompt - Sobe 1 broker com configuracao interativa"
	@echo "  clean             - Remove containers"
	@echo ""
	@echo "Exemplos:"
	@echo "  make -f Makefile.brokers run-all"
	@echo "  make -f Makefile.brokers run-broker-prompt"

build:
	@echo "Compilando imagem Docker dos brokers..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

run-all: run-broker1 run-broker2 run-broker3

run-broker1:
	docker run -d --rm \
		--name $(BROKER1_NAME) \
		-p $(BROKER1_PORT):$(BROKER1_PORT) \
		$(IMAGE_NAME) \
		-id=$(BROKER1_NAME) \
		-port=$(BROKER1_PORT)

run-broker2:
	docker run -d --rm \
		--name $(BROKER2_NAME) \
		-p $(BROKER2_PORT):$(BROKER2_PORT) \
		$(IMAGE_NAME) \
		-id=$(BROKER2_NAME) \
		-port=$(BROKER2_PORT)

run-broker3:
	docker run -d --rm \
		--name $(BROKER3_NAME) \
		-p $(BROKER3_PORT):$(BROKER3_PORT) \
		$(IMAGE_NAME) \
		-id=$(BROKER3_NAME) \
		-port=$(BROKER3_PORT)

run-broker-prompt:
	@echo "Configuracao interativa do broker"
	@echo "Pressione ENTER para usar o padrao"
	@echo ""
	@echo "Nome do broker"; \
	echo "BROKER_NAME [padrao: $(BROKER1_NAME)]:"; \
	read _BROKER_NAME; \
	_BROKER_NAME=$${_BROKER_NAME:-$(BROKER1_NAME)}; \
	echo "Porta do broker"; \
	echo "BROKER_PORT [padrao: $(BROKER1_PORT)]:"; \
	read _BROKER_PORT; \
	_BROKER_PORT=$${_BROKER_PORT:-$(BROKER1_PORT)}; \
	echo "Iniciando broker $$_BROKER_NAME na porta $$_BROKER_PORT..."; \
	docker run -it --rm \
		--name $$_BROKER_NAME \
		-p $$_BROKER_PORT:$$_BROKER_PORT \
		$(IMAGE_NAME) \
		-id=$$_BROKER_NAME \
		-port=$$_BROKER_PORT

clean:
	@echo "Limpando containers dos brokers..."
	-@docker stop $(BROKER1_NAME) $(BROKER2_NAME) $(BROKER3_NAME) 2>/dev/null || true
	-@docker rm $(BROKER1_NAME) $(BROKER2_NAME) $(BROKER3_NAME) 2>/dev/null || true