# Warzone: Coordenação Distribuída de Drones Marítimos

**PBL 2 de Redes de Computadores / Sistemas Distribuídos (UEFS)**

O **Warzone** é uma infraestrutura distribuída e resiliente desenvolvida em Go para a monitorização do Estreito de Ormuz. O sistema resolve desafios de concorrência e o problema do Ponto Único de Falha (SPOF) utilizando uma arquitetura híbrida: **MQTT na borda (Edge)** para comunicação com dispositivos IoT (sensores e drones) e o **Algoritmo de Consenso Raft no núcleo** para garantir a replicação segura do estado global numa Máquina de Estados Replicada (RSM).

## Tecnologias Utilizadas
* **Linguagem:** [Go (1.25.1)](https://go.dev/)
* **Consenso e Replicação:** [HashiCorp Raft](https://github.com/hashicorp/raft)
* **Mensageria:** [Mochi MQTT](https://github.com/mochi-mqtt/server) (Broker Embutido) e [Eclipse Paho MQTT](https://github.com/eclipse/paho.mqtt.golang) (Cliente)
* **Ambiente de Execução:** [Docker](https://www.docker.com/) e Makefiles
* **Ordenação Causal:** Relógios Lógicos de Lamport e Fila de Prioridades (Min-Heap)

---

## Estrutura

O projeto adota uma arquitetura modularizada, onde cada serviço está isolado no seu próprio diretório (`sector-manager`, `drones`, `sensors`, `test-client`). 

Para garantir a reprodutibilidade e o isolamento dos processos, todos os componentes são executados em contêineres **Docker**. Cada diretório possui um `Makefile` dedicado para facilitar a execução. Se não desejar compilar o código-fonte localmente, os Makefiles estão configurados para descarregar as imagens diretamente do Docker Hub.

**Comandos Universais:**
* `make build`: Constrói a imagem Docker do componente a partir do código-fonte local.
* `make pull`: Descarrega a imagem pré-compilada do Docker Hub (recomendado para testes rápidos).
* `make help`: Lista todos os comandos e opções disponíveis para o módulo em questão.

---

## Como utilizar os Makefiles

O projeto adota uma arquitetura onde cada serviço está isolado em sua própria pasta (`sector-manager`, `drones`, `sensors`, etc.), e cada pasta contém seu próprio `Makefile`. 

Estes comandos facilitam a criação dos componentes e a subir os contêineres Docker.

Comandos mais comuns:

- `make build`: Constrói a imagem Docker do componente.
- `make run`: Executa o container Docker recém-construído.
- `make pull`: Puxa a imagem do Dockerhub

Se não quiser baixar todo o projeto, basta apenas baixar os Makefiles correspondentes e puxar a imagem Docker. Ou, se quiser poupar este trabalho, puxar as imagens Docker diretamente do terminal e subir o contêiner informando as _flags_.

## Como subir os componentes

> [!NOTE]
> Os contêineres rodam em modo _host_ e, portanto, não precisam expor as portas.

### Sensores

Construa ou puxe a imagem do Dockerhub utilizando o comando `make pull`.
Utilize o comando `make run` para criar um sensor padrão que se conecta em "tcp://127.0.0.1:1883".

Para configurar o sensor à sua vontade, utilize `make run-prompt`. Você precisará informar:

1. O ID do sensor
2. O tipo do sensor
3. O endereço do _broker_ MQTT do setor desejado

#### Utilizando o sensor de teste

Quando o líder do Raft estiver em modo de depuração, utilize o comando `make run-lento` para criar um sensor com um ID especial. Os alertas criados por sensor possuirão um atraso intencional até que sejam adicionados à fila de prioridades na RSM.

### Drones

Construa ou puxe a imagem do Dockerhub utilizando o comando `make pull`.
Utilize o comando `make run` para criar um drone padrão que se tenta se conectar em "tcp://127.0.0.1:1883, tcp://127.0.0.1:1884 e tcp://127.0.0.1:1885".

O comando `make run-prompt` perguntará ao usuário as _flags_ de ID do drone e da lista dos _brokers_ dos setores (deve conter o endereço completo, no tipo tcp://IP:PORTA). O drone começa tentando se conectar no primeiro _broker_ da lista. 


### Gerenciadores de setor

Construa ou puxe a imagem do Dockerhub utilizando o comando `make pull`.
O Makefile dispõe de comandos para instanciar setores pré-configurados ou não. O pré-configurado cria três setores (A, B e C) e os conecta automaticamente:

```bash
make run-sectorA  # Sobe o nó do setor A
make run-sectorB  # Sobe o nó do setor B
make run-sectorC  # Sobe o nó do setor C
```

Para configurar o nó à sua vontade, use:

``` bash
make run-node-prompt
```

O comando irá pedir o valor das seguintes _flags_:

* **`NODE_ID`:** O identificador único do setor no cluster (ex: `setorA`, `setorD`). Este valor define o nome do contêiner Docker e a pasta onde os dados do Raft (logs e snapshots) serão persistidos. *(Padrão: `setorA`)*
* **`BOOTSTRAP`:** Define se este nó será o responsável por iniciar um cluster do zero (`true` ou `false`). Em um cluster novo, **apenas o primeiro nó** deve ser iniciado com `true`; os demais devem entrar como `false`. *(Padrão: `false`)*
* **`PEERS`:** Lista de endereços dos nós que já fazem parte do cluster para que este novo nó possa se juntar a eles. Deve ser informado no formato `host:porta_sinalizacao` separados por vírgula (ex: `127.0.0.1:11001,127.0.0.1:11002`). *(Padrão: vazio)*
* **`HOST`:** O endereço de IP onde o nó vai ser executado e expor seus serviços na rede (útil caso vá rodar em máquinas físicas diferentes). *(Padrão: `127.0.0.1`)*
* **`RAFT_PORT`:** A porta TCP dedicada exclusivamente à comunicação interna do algoritmo Raft (eleições e replicação contínua da Máquina de Estados). *(Padrão: `10001`)*
> [!NOTE]
> A porta de sinalização é automaticamente definida como a RAFT_PORT + 1000 (ex: 11001)
* **`BROKER_PORT`:** A porta TCP na qual o broker MQTT embutido neste setor ficará escutando. É neste endereço que os sensores e drones daquele setor irão se conectar. *(Padrão: `1883`)*
* **`DASHBOARD_PORT`:** A porta HTTP na qual o painel de visualização web (Dashboard) deste setor ficará acessível através do navegador. *(Padrão: `8081`)*

Se você quiser recomeçar/reconfigurar o cluster, deve limpar os logs/snapshots. Para isso, use:

``` bash
make clean
```
#### Modo de depuração

O setor conta com um modo debug, que traz logs e funcionalidades extras para depuração. Para utilizar, adicione ao fim de qualquer run, o comando:

``` bash
FLAGS="-debug"
```

Você pode usar `make help` dentro dos diretórios que disponibilizam essa flag para listar os comandos exatos de cada módulo.

#### Dashboard

Os gerenciadores de setor contam com uma interface em terminal. Elas rodam na máquina local (_localhost_) na porta informada na criação do contêiner.

<img width="1782" height="879" alt="image psd" src="https://github.com/user-attachments/assets/229025d1-cae8-4c62-b8dd-8d99ff34ab26" />


## Como utilizar o test-client (Simulador)

A pasta `test-client` expõe uma interface interativa de terminal. Foi programada para inserção manual de ocorrências (via broker), testes de estresse e para simular múltiplos drones virtuais entrando e saindo de serviço. Ele não é necessário para o funcionamento padrão do sistema.

Instruções:

Crie ou puxe a imagem Docker e então abra a interface de linha de comando com:
```bash
make run
```

### Configurando e usando o menu interativo

Ao subir, será solicitado o endereço do Broker na rede e sua porta (ex: `tcp://127.0.0.1:1883`, caso teste localmente no MQTT do Setor A). Assim que conectado, será impresso um menu de múltipla escolha onde você pode digitar o número referente à ação desejada.

1. Gera um alerta manual. Você pode escolher o tipo do sensor, para testar as prioridades.
2. Gera n alertas em sequência. O tipo dos sensores é aleatório.
3. Gera um alerta com atraso. Sempre é do tipo de incêndio.
> [!NOTE]
> O teste de sensor atrasado só funciona se o cliente estiver conectado ao líder, que deve estar em modo debug (FIX em breve).
4. Simula a conexão de n sensores ao _broker_. Esses sensores não geram alertas e se desconectam após 1 minuto.
5. Conecta n drones ao _broker_. Os drones são funcionais e realizam missões, se desconectando após 5 minutos.
> [!NOTE]
> Se as opções 4 e 5 forem utilizadas novamente antes dos disposiivos se desconectarem, os novos roubarão os IDs dos antigos e ocorrerá _trashing_ dos clientes (já documentado).

