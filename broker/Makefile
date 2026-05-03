IMAGE_NAME = sector-broker
NETWORK = warzone-net

# Cores para o terminal
CYAN = \033[0;36m
RESET = \033[0m

.PHONY: build setup-net run-broker1 run-broker2 run-broker3 clean

build:
	@echo "$(CYAN)Compilando imagem Docker...$(RESET)"
	docker build -t $(IMAGE_NAME) -f dockerfile ..

setup-net:
	@echo "$(CYAN)Criando rede isolada...$(RESET)"
	@docker network inspect $(NETWORK) >/dev/null 2>&1 || docker network create $(NETWORK)

run-broker1: setup-net
	@echo "$(CYAN)Iniciando Broker do Setor 1...$(RESET)"
	docker run -it --rm \
		--name broker1 \
		--network $(NETWORK) \
		$(IMAGE_NAME)

run-broker2: setup-net
	@echo "$(CYAN)Iniciando Broker do Setor 2...$(RESET)"
	docker run -it --rm \
		--name broker2 \
		--network $(NETWORK) \
		$(IMAGE_NAME)

run-broker3: setup-net
	@echo "$(CYAN)Iniciando Broker do Setor 3...$(RESET)"
	docker run -it --rm \
		--name broker3 \
		--network $(NETWORK) \
		$(IMAGE_NAME)

clean:
	@echo "$(CYAN)Limpando containers e rede...$(RESET)"
	-docker stop broker1 broker2 broker3
	-docker rm broker1 broker2 broker3
	-docker network rm $(NETWORK)
