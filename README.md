# Warzone
PBL 2 de Redes de Computadores

## Arquitetura (resumo)
- **Raft/FSM**: mantém apenas estado interno e aplicação de comandos.
- **Messaging (MQTT)**: isolado em `internal/messaging`, com reconexão automática e tolerância a falhas.
- **Handlers MQTT**: apenas traduzem mensagens de entrada em comandos para Raft.
- **Eventos de saída** (`drones/.../mission`, `sensors/.../solved`): gerados pelo estado interno e publicados fora do FSM.
