SHELL := /bin/bash

IMAGE_NAME ?= daviuefs/drones

DRONE_ID ?= drone-01
BROKERS ?= tcp://127.0.0.1:1883,tcp://127.0.0.1:1884,tcp://127.0.0.1:1885

.PHONY: help build run run-prompt clean

help:
	@echo "Targets:"
	@echo "  build       - Build da imagem Docker do drone"
	@echo "  run         - Sobe um drone com os valores padrao"
	@echo "  run-prompt  - Sobe um drone com configuracao interativa"
	@echo "  clean       - Remove o container do drone"

build:
	@echo "Compilando imagem Docker do drone..."
	docker build -t $(IMAGE_NAME) -f dockerfile ..

pull:
	@echo "Baixando imagem do Docker Hub..."
	docker pull $(IMAGE_NAME):latest

run:
	docker run -it --rm \
		--name $(DRONE_ID) \
		--network host \
		$(IMAGE_NAME) \
		-id=$(DRONE_ID) \
		-brokers=$(BROKERS)

run-go:
	@echo "Iniciando drone $(DRONE_ID) sem Docker..."
	go run . \
		-id=$(DRONE_ID) \
		-brokers=$(BROKERS)

run-prompt:
	@echo "Configuracao interativa do drone"
	@echo "Pressione ENTER para usar o padrao"
	@echo ""
	@echo "ID do drone"; \
	echo "DRONE_ID [padrao: $(DRONE_ID)]:"; \
	read _DRONE_ID; \
	_DRONE_ID=$${_DRONE_ID:-$(DRONE_ID)}; \
	echo "Lista de brokers (separados por virgula)"; \
	echo "BROKERS [padrao: $(BROKERS)]:"; \
	read _BROKERS; \
	_BROKERS=$${_BROKERS:-$(BROKERS)}; \
	echo "Iniciando drone $$_DRONE_ID..."; \
	docker run -it --rm \
		--name $$_DRONE_ID \
		--network host \
		$(IMAGE_NAME) \
		-id=$$_DRONE_ID \
		-brokers=$$_BROKERS

clean:
	@echo "Limpando container do drone..."
	-@docker stop $(DRONE_ID) 2>/dev/null || true
	-@docker rm $(DRONE_ID) 2>/dev/null || true
