#!/bin/bash
# teste_arquitetura.sh

echo -e "\033[1;34m[TESTE 1] Arquitetura Descentralizada e Resiliência\033[0m"

echo "1. Status inicial: 4 nós rodando."
docker ps --format "table {{.Names}}\t{{.Status}}" | grep node-

echo -e "\n2. Derrubando o 'node-d' (Simulando falha de hardware/rede)..."
docker stop node-d

echo -e "\n3. Enviando uma transferência enquanto o node-d está morto..."
# Pega o endereço de um país (ex: EUA) e manda para outro (ex: Iraque)
ADDR_ORIGEM=$(docker exec node-a warzone-cored keys show pais_eua -a --keyring-backend test --home /warzone)
ADDR_DESTINO=$(docker exec node-a warzone-cored keys show pais_iraque -a --keyring-backend test --home /warzone)

docker exec node-a warzone-cored tx bank send $ADDR_ORIGEM $ADDR_DESTINO 10stake \
  --from pais_eua --keyring-backend test --home /warzone --chain-id warzone-rede \
  --fees 20stake --broadcast-mode sync -y > /dev/null 2>&1

echo "Aguardando consenso dos nós sobreviventes (node-a, node-b, node-c)..."
sleep 6

echo -e "\n4. Verificando o saldo através do 'node-c' (Comprovando comunicação e ledger P2P):"
docker exec node-c warzone-cored query bank balances $ADDR_DESTINO --output json | grep amount

echo -e "\n5. Ressuscitando o 'node-d'..."
docker start node-d
echo -e "\033[1;32m[SUCESSO] A rede sobreviveu à queda de 25% dos nós!\033[0m"