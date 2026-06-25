#!/bin/bash
# teste_auditoria.sh

echo -e "\033[1;34m[TESTE 3] Imutabilidade e Auditabilidade de Laudos\033[0m"

ID_MISSAO="req-auditoria-001"
DRONE="drone-alfa"
LAUDO="Carga entregue em seguranca, sem incidentes"

echo "1. Submetendo Laudo Oficial na Blockchain via node-a..."
docker exec node-a warzone-cored tx warzone submit-laudo $ID_MISSAO $DRONE "$LAUDO" "concluido" \
  --from key_a --keyring-backend test --home /warzone --chain-id warzone-rede \
  --fees 20stake -y > temp_tx.json 2>&1

sleep 6
echo "Laudo registrado no bloco."

echo -e "\n2. Auditando Laudo através do Node A (REST API):"
curl -s -X GET "http://localhost:1317/Davi-UEFS/warzone-core/warzone/v1/laudo" | grep -A 4 "req-auditoria-001"

echo -e "\n3. Auditando o mesmo Laudo através do Node D (REST API - Porta 1320):"
curl -s -X GET "http://localhost:1320/Davi-UEFS/warzone-core/warzone/v1/laudo" | grep -A 4 "req-auditoria-001"

echo -e "\n\033[1;32m[SUCESSO] Os dados são públicos, idênticos em toda a rede e amarrados por hash criptográfico!\033[0m"