#!/bin/bash
echo -e "\033[1;34m[TESTE 5] Defesa contra Adulteração Local (Data Tampering)\033[0m"

DATA_DIR="$(eval echo ~)/warzone-data/setorD/data/application.db"

echo "1. Desligando o node-d..."
docker stop node-d
sleep 2

echo "2. Injetando corrupção no LevelDB (IAVL tree do CosmosSDK)..."
FILE=$(ls -S "$DATA_DIR"/*.ldb | head -1)
SIZE=$(wc -c < "$FILE")
MID=$((SIZE / 2))
echo "   → Arquivo alvo: $FILE"
echo "   → Tamanho: $SIZE bytes, corrompendo no byte $MID"
printf '\xDE\xAD\xBE\xEF\xDE\xAD\xBE\xEF' | dd of="$FILE" bs=1 seek=$MID conv=notrunc
echo "   → Corrupção injetada!"

echo "3. Religando node-d com dados adulterados..."
docker start node-d
sleep 6

echo "4. Lendo logs de erro do node-d..."
docker logs --tail 40 node-d 2>&1

echo ""
echo "=== FILTRANDO DETECÇÃO DE FRAUDE ==="
docker logs node-d 2>&1 | grep -iE "panic|corrupt|mismatch|apphash|wrong block|invalid|ERR"
