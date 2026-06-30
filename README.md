# 🚁 Warzone – Economia e Auditoria de Guerra (AppChain)

## 📖 Sobre

O **Warzone** é um sistema distribuído para gerenciamento do despacho de drones autônomos em zonas de conflito.

Nesta versão, a arquitetura foi migrada de um modelo baseado em **Raft (CFT)** para uma **AppChain privada** desenvolvida com o **Cosmos SDK** e utilizando o **CometBFT** como mecanismo de consenso.

Essa abordagem permite que múltiplas organizações operem uma blockchain permissionada, garantindo integridade dos dados, auditoria permanente e descentralização da tomada de decisões.

### Principais Características

- ✅ **Prevenção contra duplo gasto**
  - Uma companhia/país somente pode requisitar recursos caso possua saldo suficiente de tokens (`stake`).

- 🔒 **Imutabilidade dos registros**
  - Todas as operações financeiras, missões e eventos ficam armazenados permanentemente na blockchain através da **IAVL Merkle Tree**.

- 🌐 **Descentralização**
  - A rede é composta por múltiplos validadores conectados via protocolo **P2P Gossip**, utilizando consenso **Byzantine Fault Tolerant (CometBFT)**.

- ⚡ **Alta disponibilidade**
  - Os clientes implementam **failover automático**, buscando outro nó da rede caso o validador atual fique indisponível.

---

# 🏗️ Estrutura do Projeto

| Diretório | Descrição |
|-----------|-----------|
| **warzone-core/** | Blockchain (AppChain Cosmos SDK). Contém a máquina de estados, módulos, lógica de consenso e regras de negócio. |
| **sector-manager/** | Hub tático responsável pela comunicação entre MQTT e Blockchain (REST/RPC). Também hospeda o Dashboard Web. |
| **drones/** | Cliente IoT que simula drones autônomos, consumo de bateria, registro e execução de missões. |
| **sensors/** | Sensores IoT responsáveis por publicar eventos e incidentes via MQTT. |
| **test-client/** | Ferramenta para geração de eventos, testes de estresse e criação dinâmica de sensores e drones. |
| **painel/** | Painel global de auditoria que consulta diretamente os saldos da blockchain com suporte a failover. |
| **testes/** | Scripts automatizados para validar segurança, integridade e resiliência da blockchain. |

---

# ⚙️ Pré-requisitos

- Go **1.21+**
- Docker
- Docker Compose
- Make
- Ignite CLI *(opcional, apenas para modificar a blockchain)*

---

# 🚀 Executando o Sistema

## 1. Imagens (Docker) e Makefile

Para fazer qualquer operação na rede é necessário as imagens dos containers e do Makefile.

```bash
docker pull daviuefs/core
```

Baixe o arquivo Makefile.docker diretamente do repositório https://github.com/Davi-UEFS/Warzone.git. O arquivo se encontra em Warzone/warzone-core.
Obs: Caso não possua nenhum outro Makefile na pasta onde utilizará o Makefile.docker, o sufixo .docker pode ser removido e então poderá usar o arquivo como um Makefile comum. Do contrário, após o comando make insira "-f Makefile.docker". Os exemplos nesse README não estarão com essa extensão.

```bash
make -f Makefile.docker ...
```

## 2. Preparar a Blockchain

Antes de iniciar a rede é necessário gerar:

- contas das companhias;
- chaves dos validadores;
- arquivos de gênese;
- configuração dos setores.

```bash
make prepare
```

Serão criadas 4 pastas: setorA, setorB, setorC, setorD. Cada pasta possui as configurações necessárias (contas, chaves, gênese e etc.).

---


## 3. Iniciar a Rede Blockchain

### Opção A — Ambiente Local

Executa quatro validadores utilizando Docker.

```bash
make run-local
```

Nesta opção não é necessário nenhuma configuração adicional

---

### Opção B — Rede Local (LAN)

Cada computador representa um país/validador. É necessário que a pasta do setor (setorA, setorB, ...) esteja no computador que executará o respectivo setor.

Para isso, crie uma pasta no diretório raiz do usuário chamada "warzone-data".

```bash
cd
mkdir warzone-data
```
Em seguida, mova a pasta do setor que irá ser inicializado para o diretório criado.

### PC Master (Gênese)

```bash
make run-lan-master
```
Obs: o PC Master é obrigatoriamente o setorA.

Ao iniciar serão exibidos:

- Node ID
- Endereço IP

Essas informações deverão ser utilizadas pelos demais nós.

---

### Demais Computadores

```bash
make run-lan-node
```

Será solicitado:

- Folder: Pasta do setor (ex.: setorA, setorB, ...)
- ID do peer: ID gerado pelo PC Master
- IP do peer: Endereço IP do Master (ex.: 192.xxx.xx.xx)

Todos os serviços REST e RPC já são expostos automaticamente para a rede local.

---

# 🌐 Dashboard Tático

Entre na pasta:

```bash
cd sector-manager
```

O Dashboard ficará disponível em:

```
http://localhost:8080
```

É possível informar múltiplos nós separados por vírgula para ativar o mecanismo de **failover automático**.

Também é possível sobrescrever variáveis como:

- `PORT`
- `SECTOR_ID`
- `BLOCKCHAIN_URL`

---

# 📡 Simulação de Eventos

Entre em:

```bash
cd test-client
```

Execute:

```bash
go run main.go
```

O cliente permite:

- criar sensores dinamicamente;
- registrar drones;
- enviar alertas em lote;
- gerar testes de estresse.

Durante a execução será solicitado um número inicial para geração dos IDs, evitando colisões no estado da blockchain.

---

# 💰 Painel Global de Auditoria

Entre em:

```bash
cd painel
```

Execute:

```bash
make run
```

O painel consulta diretamente a API REST da blockchain.

---

# 🛡️ Ferramentas de Auditoria

A pasta `testes/` contém scripts para demonstrar a segurança e resiliência da blockchain.

### Teste de Duplo Gasto

```bash
./testes/teste_duplo_gasto.sh
```

Envia múltiplas transações concorrentes tentando gastar o mesmo saldo.

Resultado esperado:

- apenas uma transação será aceita;
- as demais serão rejeitadas pela mempool ou durante o consenso.

---

### Teste de Adulteração

```bash
./testes/teste_adulteracao.sh
```

Corrompe propositalmente o banco de dados interno de um nó.

Resultado esperado:

- quebra do hash da Merkle Tree;
- isolamento automático do nó pelos demais validadores.

---

### Teste de Alocação Simultânea

```bash
./testes/teste_alocacao.sh
```

Tenta atribuir o mesmo drone para duas missões simultaneamente.

Resultado esperado:

- apenas uma alocação será validada;
- a execução atômica da blockchain garante exclusividade.

---

### Teste de Auditoria

```bash
./testes/teste_auditoria.sh
```

Extrai uma transação validada e verifica:

- assinatura digital;
- chave pública;
- autenticidade da operação.

Demonstrando o uso de **Criptografia de Curva Elíptica (ECDSA)** para autenticação das transações.

---

# 🔐 Tecnologias Utilizadas

- Cosmos SDK
- CometBFT
- IAVL Tree
- Go
- Docker
- MQTT
- REST API
- P2P Gossip Protocol
- Blockchain Permissionada

---

# 📌 Funcionalidades

- Blockchain privada baseada em Cosmos SDK
- Consenso BFT com CometBFT
- Ledger distribuído
- Auditoria imutável
- Gerenciamento de drones
- Tokens para alocação de recursos
- Dashboard Web
- Comunicação MQTT
- Failover automático
- Testes automatizados de segurança
- Simulação de ataques e estresse
```
