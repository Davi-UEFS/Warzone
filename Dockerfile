# 1. Estágio de Build
FROM golang:1.22-alpine AS builder

# Instala dependências de compilação
RUN apk add --no-cache make gcc musl-dev git

# Instala o Ignite CLI (versão compatível com a do seu lab)
RUN go install github.com/ignite/cli/v28@latest

WORKDIR /app

# Copia os arquivos de dependência
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código fonte
COPY . .

# Roda o build como se estivesse no seu PC (ignite chain build)
WORKDIR /app/warzone-core
RUN ignite chain build

# 2. Estágio Final (Imagem pronta para rodar)
FROM alpine:3.19
RUN apk add --no-cache ca-certificates libc6-compat

# Copia apenas o binário gerado pelo build acima
COPY --from=builder /app/warzone-core/warzone-cored /usr/local/bin/warzone-cored

# Expõe as portas de consenso e API
EXPOSE 26656 26657 1317 9090

ENTRYPOINT ["warzone-cored"]