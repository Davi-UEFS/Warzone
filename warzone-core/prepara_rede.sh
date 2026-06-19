#!/bin/bash
BIN="$HOME/go/bin/warzone-cored"

# ==========================================
# DEFINIÇÃO DO DIRETÓRIO EXTERNO DOS DADOS
# ==========================================
# Todos os dados serão gerados fora do repositório do código
BASE_DIR="$HOME/warzone-data" 

echo "Limpando testes anteriores..."
rm -rf $BASE_DIR/setorA $BASE_DIR/setorB $BASE_DIR/setorC $BASE_DIR/setorD

# Garante que a pasta pai exista
mkdir -p $BASE_DIR

echo "1. Inicializando os 4 setores..."
$BIN init setor-a --chain-id warzone-rede --home $BASE_DIR/setorA > /dev/null 2>&1
$BIN init setor-b --chain-id warzone-rede --home $BASE_DIR/setorB > /dev/null 2>&1
$BIN init setor-c --chain-id warzone-rede --home $BASE_DIR/setorC > /dev/null 2>&1
$BIN init setor-d --chain-id warzone-rede --home $BASE_DIR/setorD > /dev/null 2>&1

echo "2. Criando as carteiras de cada setor..."
$BIN keys add key_a --keyring-backend test --home $BASE_DIR/setorA > /dev/null 2>&1
$BIN keys add key_b --keyring-backend test --home $BASE_DIR/setorB > /dev/null 2>&1
$BIN keys add key_c --keyring-backend test --home $BASE_DIR/setorC > /dev/null 2>&1
$BIN keys add key_d --keyring-backend test --home $BASE_DIR/setorD > /dev/null 2>&1

# Extraindo os endereços gerados
ADDR_A=$($BIN keys show key_a -a --keyring-backend test --home $BASE_DIR/setorA)
ADDR_B=$($BIN keys show key_b -a --keyring-backend test --home $BASE_DIR/setorB)
ADDR_C=$($BIN keys show key_c -a --keyring-backend test --home $BASE_DIR/setorC)
ADDR_D=$($BIN keys show key_d -a --keyring-backend test --home $BASE_DIR/setorD)

echo "3. Adicionando fundos iguais no Gênese Base..."
$BIN genesis add-genesis-account $ADDR_A 10000000stake --home $BASE_DIR/setorA
$BIN genesis add-genesis-account $ADDR_B 10000000stake --home $BASE_DIR/setorA
$BIN genesis add-genesis-account $ADDR_C 10000000stake --home $BASE_DIR/setorA
$BIN genesis add-genesis-account $ADDR_D 10000000stake --home $BASE_DIR/setorA

echo "10. Criando carteiras dos países (uma vez, no Setor A)..."
PAISES=("iraque" "eua" "arabia-saudita" "siria" "afeganistao")

echo "{" > $BASE_DIR/setorA/paises.json
PRIMEIRO=true

for PAIS in "${PAISES[@]}"; do
    MNEMONIC=$($BIN keys add "pais_${PAIS}" \
        --keyring-backend test \
        --home $BASE_DIR/setorA \
        --output json 2>&1 | python3 -c "import sys,json; print(json.load(sys.stdin)['mnemonic'])")

    ADDR=$($BIN keys show "pais_${PAIS}" -a --keyring-backend test --home $BASE_DIR/setorA)

    $BIN genesis add-genesis-account $ADDR 5000000stake --home $BASE_DIR/setorA

    for SETOR in setorB setorC setorD; do
        echo "$MNEMONIC" | $BIN keys add "pais_${PAIS}" \
            --keyring-backend test \
            --home $BASE_DIR/$SETOR \
            --recover > /dev/null 2>&1
    done

    if [ "$PRIMEIRO" = true ]; then PRIMEIRO=false; else echo "," >> $BASE_DIR/setorA/paises.json; fi
    echo "  \"$PAIS\": \"$ADDR\"" >> $BASE_DIR/setorA/paises.json
done

echo "}" >> $BASE_DIR/setorA/paises.json
echo "[OK] paises.json gerado."

cp $BASE_DIR/setorA/paises.json $BASE_DIR/setorB/paises.json
cp $BASE_DIR/setorA/paises.json $BASE_DIR/setorC/paises.json
cp $BASE_DIR/setorA/paises.json $BASE_DIR/setorD/paises.json

echo "4. Distribuindo o Gênese Base para que todos possam apostar..."
cp $BASE_DIR/setorA/config/genesis.json $BASE_DIR/setorB/config/genesis.json
cp $BASE_DIR/setorA/config/genesis.json $BASE_DIR/setorC/config/genesis.json
cp $BASE_DIR/setorA/config/genesis.json $BASE_DIR/setorD/config/genesis.json

echo "5. Cada setor gera sua aposta de 25% (2.500.000 stake)..."
$BIN genesis gentx key_a 2500000stake --chain-id warzone-rede --keyring-backend test --home $BASE_DIR/setorA > /dev/null 2>&1
$BIN genesis gentx key_b 2500000stake --chain-id warzone-rede --keyring-backend test --home $BASE_DIR/setorB > /dev/null 2>&1
$BIN genesis gentx key_c 2500000stake --chain-id warzone-rede --keyring-backend test --home $BASE_DIR/setorC > /dev/null 2>&1
$BIN genesis gentx key_d 2500000stake --chain-id warzone-rede --keyring-backend test --home $BASE_DIR/setorD > /dev/null 2>&1

echo "6. Agrupando todas as apostas no Setor A..."
mkdir -p $BASE_DIR/setorA/config/gentx/
cp $BASE_DIR/setorB/config/gentx/*.json $BASE_DIR/setorA/config/gentx/
cp $BASE_DIR/setorC/config/gentx/*.json $BASE_DIR/setorA/config/gentx/
cp $BASE_DIR/setorD/config/gentx/*.json $BASE_DIR/setorA/config/gentx/

echo "7. Consolidando o Gênese FINAL com os 4 validadores..."
$BIN genesis collect-gentxs --home $BASE_DIR/setorA > /dev/null 2>&1

echo "8. Distribuindo o Gênese FINAL e pronto para todos os nós..."
cp $BASE_DIR/setorA/config/genesis.json $BASE_DIR/setorB/config/genesis.json
cp $BASE_DIR/setorA/config/genesis.json $BASE_DIR/setorC/config/genesis.json
cp $BASE_DIR/setorA/config/genesis.json $BASE_DIR/setorD/config/genesis.json

echo "9. Pegando o endereço de conexão P2P..."
NODE_A_ID=$($BIN tendermint show-node-id --home $BASE_DIR/setorA)

echo "======================================================"
echo "REDE BFT BALANCEADA PRONTA (4 NÓS - 25% DE PODER CADA)"
echo "COPIE O CÓDIGO ABAIXO PARA CONECTAR OS NÓS:"
echo "$NODE_A_ID@127.0.0.1:26656"
echo "======================================================"