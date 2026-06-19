SHELL := /bin/bash

# Variaveis Padrao
IMAGE_NAME ?= daviuefs/client-transfer
CONTAINER_NAME ?= transfer-client

BROKER_ADDR ?= tcp://127.0.0.1:1883
FROM_ALIAS ?= Setor-A
TO_ADDR ?= cosmos1enderecoficticio1234567890
AMOUNT ?= 100stake

.PHONY: help build run run-go run-prompt clean

help:
	@echo "Targets:"
	@echo "  build        - Build da imagem Docker do cliente de transferencia"
	@echo "  run          - Executa a transferencia com os valores padrao via Docker"
	@echo "  run-go       - Executa a transferencia nativamente (precisa do Go instalado)"
	@echo "  run-prompt   - Executa com configuracao interativa (Recomendado para a apresentacao)"
	@echo "  clean        - Remove containers pendentes do cliente"

build:
	@echo "Compilando imagem Docker do cliente de transferencia..."
	docker build -t $(IMAGE_NAME) -f Dockerfile .

run:
	@echo "Iniciando transferencia via Docker..."
	docker run -it --rm \
		--name $(CONTAINER_NAME) \
		--network host \
		$(IMAGE_NAME) \
		-broker=$(BROKER_ADDR) \
		-from=$(FROM_ALIAS) \
		-to=$(TO_ADDR) \
		-amount=$(AMOUNT)

run-go:
	@echo "Iniciando transferencia sem Docker..."
	go run . \
		-broker=$(BROKER_ADDR) \
		-from=$(FROM_ALIAS) \
		-to=$(TO_ADDR) \
		-amount=$(AMOUNT)

run-prompt:
	@echo "========================================"
	@echo "  CONFIGURACAO DE TRANSFERENCIA (MQTT)  "
	@echo "========================================"
	@echo "Pressione ENTER para usar o valor padrao"
	@echo ""
	@echo "Endereco do broker MQTT"; \
	echo "BROKER_ADDR [padrao: $(BROKER_ADDR)]:"; \
	read _BROKER_ADDR; \
	_BROKER_ADDR=$${_BROKER_ADDR:-$(BROKER_ADDR)}; \
	echo ""; \
	echo "Origem dos Fundos (Apelido ou Pais, ex: Brasil, EUA, Setor-A)"; \
	echo "FROM_ALIAS [padrao: $(FROM_ALIAS)]:"; \
	read _FROM_ALIAS; \
	_FROM_ALIAS=$${_FROM_ALIAS:-$(FROM_ALIAS)}; \
	echo ""; \
	echo "Destino dos Fundos (Endereco bech32: cosmos1...)"; \
	echo "TO_ADDR [padrao: $(TO_ADDR)]:"; \
	read _TO_ADDR; \
	_TO_ADDR=$${_TO_ADDR:-$(TO_ADDR)}; \
	echo ""; \
	echo "Quantidade e Moeda (ex: 50stake, 1000token)"; \
	echo "AMOUNT [padrao: $(AMOUNT)]:"; \
	read _AMOUNT; \
	_AMOUNT=$${_AMOUNT:-$(AMOUNT)}; \
	echo ""; \
	echo "Enviando solicitacao de transferencia..."; \
	docker run -it --rm \
		--name $(CONTAINER_NAME) \
		--network host \
		$(IMAGE_NAME) \
		-broker=$$_BROKER_ADDR \
		-from=$$_FROM_ALIAS \
		-to=$$_TO_ADDR \
		-amount=$$_AMOUNT

clean:
	@echo "Limpando container do cliente de transferencia..."
	-@docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-@docker rm $(CONTAINER_NAME) 2>/dev/null || true