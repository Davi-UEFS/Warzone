# Warzone Project
PBL 2 de Redes de Computadores

Este é um documento temporário com instruções básicas para compilar e executar os módulos do sistema. O documento definitivo será disponibilizado futuramente.

## Como utilizar os Makefiles

O projeto adota uma arquitetura onde cada serviço está isolado em sua própria pasta (`sector-manager`, `drones`, `sensors`, etc.), e cada pasta contém seu próprio `Makefile`. 

Estes comandos facilitam a criação dos componentes e a subir os contêineres Docker.

Comandos mais comuns:

- `make build`: Constrói a imagem Docker do componente.
- `make run`: Executa o container Docker recém-construído.
- `make pull`: Puxa a imagem do Dockerhub

Se não quiser baixar toda o projeto, basta apenas baixar os Makefiles correspondentes e puxar a imagem Docker.

### Exemplo (Sector Manager)

No caso do `sector-manager`, ele representa os nós de um cluster Raft. O Makefile dispõe de comandos para instanciar 
setores pré-configurados ou não. O pré-configurado cria três setores e os conecta automaticamente:

```bash
make build        # Alternativamente, pode usar make pull
make run-sectorA  # Sobe o nó do setor A
make run-sectorB  # Sobe o nó do setor B
make run-sectorC  # Sobe o nó do setor C
```

Para configurar o nó à sua vontade, use:

``` bash
make run-node-prompt
```

Se você quiser recomeçar/reconfigurar o cluster, deve limpar os logs/snapshots. Para isso, use:

``` bash
make clean
```

O setor conta com um modo debug, que traz logs e funcionalidades extras para depuração. Para utilizar, adicione ao fim de qualquer run, o comando:

``` bash
FLAGS="-debug"
```

Você pode usar `make help` dentro dos diretórios que disponibilizam essa flag para listar os comandos exatos de cada módulo.

## Como utilizar o test-client (Simulador)

A pasta `test-client` expõe uma interface interativa de terminal. Foi programada para inserção manual de ocorrências (via broker), testes de estresse e para simular múltiplos drones virtuais entrando e saindo de serviço.

Instruções:

0. Se você já possui a imagem, pule para o passo 3.

1. Acesse o diretório do cliente:
```bash
cd test-client
```

2. Realize o build:
```bash
make build
```

3. Abra a interface de linha de comando com:
```bash
make run
```

### Configurando e usando o menu interativo

1. Ao subir, será solicitado o endereço do Broker na rede e sua porta (ex: `tcp://127.0.0.1:1883`, caso teste localmente no MQTT do Setor A).
2. Assim que conectado, será impresso um menu de múltipla escolha onde você pode digitar o número referente à ação desejada.
3. Você pode inserir um alerta isolado ou gerar lotes, conectar drones, etc.

> [!NOTE]
> O teste de sensor atrasado só funciona se o líder estiver em modo debug.
