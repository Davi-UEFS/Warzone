IMAGE_NAME = sector-manager
NETWORK = warzone-net
BROKER1_ADDR = tcp://broker1:1883
BROKER2_ADDR = tcp://broker2:1883
BROKER3_ADDR = tcp://broker3:1883

# Cores para o terminal
CYAN = \033[0;36m
RESET = \033[0m

.PHONY: build setup-net run-node1 run-node2 run-node3 clean

build:
	@echo "$(CYAN)Compilando imagem Docker...$(RESET)"
	docker build -t $(IMAGE_NAME) -f dockerfile ..

setup-net:
	@echo "$(CYAN)Criando rede isolada...$(RESET)"
	@docker network inspect $(NETWORK) >/dev/null 2>&1 || docker network create $(NETWORK)

# Nó 1: Liderança/Bootstrap
run-node1: setup-net
	@echo "$(CYAN)Iniciando Nó 1 (Bootstrap)...$(RESET)"
	docker run -it --rm \
		--name node1 \
		--network $(NETWORK) \
		-v $(shell pwd)/data/node1:/data/node1 \
		$(IMAGE_NAME) \
		-id=node1 -raft=node1:10001 -dir=/data/node1 -bootstrap=true -sig=node1:9123 -broker=$(BROKER1_ADDR)

# Nó 2: Seguidor (Aponta para o node1)
run-node2: setup-net
	@echo "$(CYAN)Iniciando Nó 2...$(RESET)"
	docker run -it --rm \
		--name node2 \
		--network $(NETWORK) \
		-v $(shell pwd)/data/node2:/data/node2 \
		$(IMAGE_NAME) \
		-id=node2 -raft=node2:10001 -dir=/data/node2 -peers=node1 -sig=node2:9123 -broker=$(BROKER2_ADDR)

# Nó 3: Seguidor (Aponta para node1 e node2)
run-node3: setup-net
	@echo "$(CYAN)Iniciando Nó 3...$(RESET)"
	docker run -it --rm \
		--name node3 \
		--network $(NETWORK) \
		-v $(shell pwd)/data/node3:/data/node3 \
		$(IMAGE_NAME) \
		-id=node3 -raft=node3:10001 -dir=/data/node3 -peers=node1,node2 -sig=node3:9123 -broker=$(BROKER3_ADDR)

clean:
	@echo "$(CYAN)Limpando containers e dados...$(RESET)"
	-docker stop node1 node2 node3
	-docker rm node1 node2 node3
	-docker network rm $(NETWORK)
	rm -rf data/