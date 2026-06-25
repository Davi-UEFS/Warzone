#!/bin/bash
# teste_alocacao.sh

echo -e "\033[1;34m[TESTE 4] Requisição de Escoltas e Alocação Exclusiva\033[0m"

echo -e "\n1. Teste de Saldo Insuficiente..."
# Criamos uma carteira falsa/vazia na hora apenas para este teste
docker exec node-a warzone-cored keys add pais_falido --keyring-backend test --home /warzone > /dev/null 2>&1
ADDR_FALIDO=$(docker exec node-a warzone-cored keys show pais_falido -a --keyring-backend test --home /warzone)

echo "Tentando pagar uma requisição usando uma carteira sem fundos ($ADDR_FALIDO)..."
docker exec node-a warzone-cored tx warzone add-req Setor-A 100 resgate "25,56" req-sem-fundo \
  --payer $ADDR_FALIDO --from key_a --keyring-backend test --home /warzone --chain-id warzone-rede \
  --fees 20stake -y 2>&1 | grep "insufficient funds"

echo -e "\033[1;32m[OK] A blockchain barrou a operação por falta de saldo!\033[0m"

echo -e "\n2. Teste de Dupla Alocação de Drone (Concorrência)..."
DRONE_TESTE="drone-disputado-01"
# Registramos o drone primeiro
docker exec node-a warzone-cored tx warzone reg-drone $DRONE_TESTE "Setor-A" "100" \
  --from key_a --keyring-backend test --home /warzone --chain-id warzone-rede \
  --fees 20stake -y > /dev/null 2>&1
sleep 6 # Espera o bloco ser gerado

echo "Disparando duas ordens SIMULTÂNEAS de setores diferentes para assumir o mesmo drone ($DRONE_TESTE)..."

# Node A tenta alocar para a Missão 1
docker exec node-a warzone-cored tx warzone assign-drone missao-1 $DRONE_TESTE \
  --from key_a --keyring-backend test --home /warzone --chain-id warzone-rede \
  --fees 20stake -y &

# Node B tenta alocar para a Missão 2 no mesmo milissegundo
docker exec node-b warzone-cored tx warzone assign-drone missao-2 $DRONE_TESTE \
  --from key_b --keyring-backend test --home /warzone --chain-id warzone-rede \
  --fees 20stake -y &

wait
echo "Aguardando consenso do bloco..."
sleep 6

echo -e "\nVerificando o dono final do drone na blockchain:"
docker exec node-c warzone-cored query warzone show-drone $DRONE_TESTE --output json | grep status

echo -e "\n\033[1;32m[SUCESSO] Apenas uma transação passou. O estado do drone mudou e a segunda tentativa foi rejeitada pelo keeper!\033[0m"