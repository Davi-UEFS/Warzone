SHELL := /bin/bash

IMAGE_NAME ?= warzone-sensor

SENSOR_ID ?= sensor-01
SENSOR_TYPE ?= temperature
BROKER_ADDR ?= tcp://127.0.0.1:1883

.PHONY: help build run run-prompt clean

help:
	@echo "Targets:"
	@echo "  build       - Build da imagem Docker do sensor"
	@echo "  run         - Sobe um sensor com os valores padrao"
	@echo "  run-prompt  - Sobe um sensor com configuracao interativa"
	@echo "  clean       - Remove o container do sensor"

build:
	@echo "Compilando imagem Docker do sensor..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

run:
	docker run -it --rm \
		--name $(SENSOR_ID) \
		--network host \
		$(IMAGE_NAME) \
		-id=$(SENSOR_ID) \
		-type=$(SENSOR_TYPE) \
		-broker=$(BROKER_ADDR)

run-prompt:
	@echo "Configuracao interativa do sensor"
	@echo "Pressione ENTER para usar o padrao"
	@echo ""
	@echo "ID do sensor"; \
	echo "SENSOR_ID [padrao: $(SENSOR_ID)]:"; \
	read _SENSOR_ID; \
	_SENSOR_ID=$${_SENSOR_ID:-$(SENSOR_ID)}; \
	echo "Tipo do sensor"; \
	echo "SENSOR_TYPE [padrao: $(SENSOR_TYPE)]:"; \
	read _SENSOR_TYPE; \
	_SENSOR_TYPE=$${_SENSOR_TYPE:-$(SENSOR_TYPE)}; \
	echo "Endereco do broker MQTT"; \
	echo "BROKER_ADDR [padrao: $(BROKER_ADDR)]:"; \
	read _BROKER_ADDR; \
	_BROKER_ADDR=$${_BROKER_ADDR:-$(BROKER_ADDR)}; \
	echo "Iniciando sensor $$_SENSOR_ID..."; \
	docker run -it --rm \
		--name $$_SENSOR_ID \
		--network host \
		$(IMAGE_NAME) \
		-id=$$_SENSOR_ID \
		-type=$$_SENSOR_TYPE \
		-broker=$$_BROKER_ADDR

clean:
	@echo "Limpando container do sensor..."
	-@docker stop $(SENSOR_ID) 2>/dev/null || true
	-@docker rm $(SENSOR_ID) 2>/dev/null || true
