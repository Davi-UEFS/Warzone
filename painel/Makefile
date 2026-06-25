# Variáveis padrão
APP_NAME=painel-saldos
GO_FILE=painel.go # Se guardou o ficheiro com outro nome, ex: painel.go, altere aqui

.PHONY: help run build clean

# Mostra o menu de ajuda por padrão
help:
	@echo "\033[1;34m======================================================\033[0m"
	@echo " Painel Global de Saldos - Comandos Disponíveis"
	@echo "\033[1;34m======================================================\033[0m"
	@echo "  make run    - Executa o painel diretamente no terminal (Recomendado)"
	@echo "  make build  - Compila o código num executável (.exe ou binário Linux)"
	@echo "  make clean  - Remove o ficheiro executável gerado"
	@echo ""

# Executa o código em tempo real
run:
	@echo "\033[1;32mConsultando a Blockchain...\033[0m"
	@go run $(GO_FILE)

# Compila o programa
build:
	@echo "\033[1;33mCompilando o executável $(APP_NAME)...\033[0m"
	@go build -o $(APP_NAME) $(GO_FILE)

# Limpa os binários gerados
clean:
	@echo "\033[1;31mRemovendo ficheiros compilados...\033[0m"
	@rm -f $(APP_NAME)