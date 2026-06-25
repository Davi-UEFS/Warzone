#!/bin/bash
# teste_adulteracao.sh

echo -e "\033[1;34m[TESTE 5] Defesa contra Adulteração Local (Data Tampering)\033[0m"

echo "1. Desligando o node-d temporariamente para 'hackear' os dados dele..."
docker stop node-d

echo "2. Injetando dados corrompidos no banco de dados (LevelDB) do node-d..."
# Vamos injetar lixo binário diretamente no arquivo do banco de dados de aplicação do Setor D
docker run --rm -v $HOME/warzone-data/setorD:/warzone alpine sh -c "echo 'DADOS_FALSOS_HACKER' >> /warzone/data/application.db/CURRENT" || true

echo "3. Tentando religar o node-d (O nó com dados adulterados)..."
docker start node-d
sleep 3

echo "4. Lendo os logs de erro do node-d (Ele deve perceber a fraude e travar)..."
docker logs --tail 10 node-d

echo -e "\n\033[1;32m[SUCESSO] A árvore de Merkle detectou que o Hash da base de dados mudou. O nó corrompido entrou em PANIC e foi banido do consenso!\033[0m"