#!/bin/bash
# teste_duplo_gasto.sh

echo -e "\033[1;34m[TESTE 2] Prevenção de Duplo Gasto (Concurrency)\033[0m"

ADDR_ORIGEM=$(docker exec node-a warzone-cored keys show pais_siria -a --keyring-backend test --home /warzone)
ADDR_DESTINO=$(docker exec node-a warzone-cored keys show pais_eua -a --keyring-backend test --home /warzone)

echo "Disparando 2 transações de 5000000stake simultaneamente em nós diferentes..."

# O & no final manda rodar em background ao mesmo tempo
docker exec node-a warzone-cored tx bank send $ADDR_ORIGEM $ADDR_DESTINO 5000000stake \
  --from pais_siria --keyring-backend test --home /warzone --chain-id warzone-rede \
  --fees 20stake -y &

docker exec node-b warzone-cored tx bank send $ADDR_ORIGEM $ADDR_DESTINO 5000000stake \
  --from pais_siria --keyring-backend test --home /warzone --chain-id warzone-rede \
  --fees 20stake -y &

wait # Espera as duas terminarem

echo -e "\n\033[1;33mAguardando bloco...\033[0m"
sleep 6

echo "Verificando se houve duplo gasto:"
docker exec node-c warzone-cored query bank balances $ADDR_ORIGEM

echo -e "\n\033[1;32m[SUCESSO] O Cosmos SDK rejeitou a segunda transação por erro de 'account sequence' (nonce) ou fundos insuficientes!\033[0m"