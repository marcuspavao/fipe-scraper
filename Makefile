-include .env
export

BINARY         := fipe-scraper
BINARY_MIGRATE := fipe-migrate
BUILD_DIR      := ./build

.PHONY: help build build-migrate test deploy build-deploy docker-build docker-up docker-down run start logs stop migrate migrate-status migrate-down migrate-vps clean

help:
	@echo "🚗 FIPE Scraper - Comandos disponíveis:"
	@echo "  make build         Compila os binários do scraper e migrate para Linux (amd64)"
	@echo "  make test          Roda os testes unitários"
	@echo "  make docker-build  Gera a imagem Docker"
	@echo "  make docker-up     Inicia os containers (MongoDB + Migrate + Scraper)"
	@echo "  make docker-down   Para e remove os containers"
	@echo "  make deploy        Envia os binários e arquivos de serviço para a VPS"
	@echo "  make build-deploy  Compila e faz o deploy em um único comando"
	@echo "  make start         Inicia o scraper na VPS via nohup"
	@echo "  make logs          Acompanha o log do scraper na VPS"
	@echo "  make stop          Para o scraper na VPS"

## Executa todos os testes unitários
test:
	@echo "🧪 Executando testes..."
	go test -v ./...

## Compila o scraper para Linux (AMD64)
build:
	@echo "🔨 Compilando scraper para Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY) ./cmd/scraper
	@echo "✅ $(BUILD_DIR)/$(BINARY) gerado"

## Compila o migrate para Linux (AMD64)
build-migrate:
	@echo "🔨 Compilando migrate para Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_MIGRATE) ./cmd/migrate
	@echo "✅ $(BUILD_DIR)/$(BINARY_MIGRATE) gerado"

## Gera imagens docker
docker-build:
	@echo "🐳 Criando imagem Docker..."
	docker compose build

## Roda via Docker Compose
docker-up:
	@echo "🚀 Subindo stack com Docker Compose..."
	docker compose up -d --build

## Para containers Docker
docker-down:
	@echo "🛑 Parando containers Docker..."
	docker compose down

## Faz o deploy do scraper + migrate + configs para a VPS
deploy:
	@echo "🚀 Enviando binários e arquivos para $(VPS_USER)@$(VPS_HOST):$(VPS_PATH)..."
	@read -sp "Senha SSH: " PASS; echo; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) "mkdir -p $(VPS_PATH)"; \
	sshpass -p "$$PASS" scp -o StrictHostKeyChecking=no $(BUILD_DIR)/$(BINARY) $(VPS_USER)@$(VPS_HOST):$(VPS_PATH)/$(BINARY); \
	sshpass -p "$$PASS" scp -o StrictHostKeyChecking=no $(BUILD_DIR)/$(BINARY_MIGRATE) $(VPS_USER)@$(VPS_HOST):$(VPS_PATH)/$(BINARY_MIGRATE); \
	if [ -f fipe-scraper.service ]; then \
		sshpass -p "$$PASS" scp -o StrictHostKeyChecking=no fipe-scraper.service $(VPS_USER)@$(VPS_HOST):$(VPS_PATH)/fipe-scraper.service; \
	fi; \
	if [ -f .env ]; then \
		sshpass -p "$$PASS" scp -o StrictHostKeyChecking=no .env $(VPS_USER)@$(VPS_HOST):$(VPS_PATH)/.env; \
	fi
	@echo "✅ Deploy concluído!"

## Build (scraper + migrate) + Deploy em um único comando
build-deploy: test build build-migrate deploy

## Roda o scraper em background na VPS via nohup
start:
	@read -sp "Senha SSH: " PASS; echo; \
	read -p "Flags (ex: -tabelas=10 -marcas=5): " FLAGS; \
	echo "🚀 Iniciando scraper na VPS em background..."; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"cd $(VPS_PATH) && nohup ./$(BINARY) $$FLAGS > scraper.log 2>&1 & echo \"PID: \$$!\""
	@echo "✅ Scraper rodando! Use 'make logs' para acompanhar."

## Acompanha o log do scraper em tempo real na VPS
logs:
	@read -sp "Senha SSH: " PASS; echo; \
	echo "📋 Acompanhando log... (Ctrl+C para sair)"; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"tail -f $(VPS_PATH)/scraper.log"

## Para o scraper em execução na VPS
stop:
	@read -sp "Senha SSH: " PASS; echo; \
	echo "🛑 Parando scraper na VPS..."; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"pkill -f $(BINARY) && echo '✅ Parado.' || echo '⚠️  Nenhum processo encontrado.'"

## Roda o scraper interativamente na VPS
run:
	@read -sp "Senha SSH: " PASS; echo; \
	read -p "Flags (ex: -tabelas=10 -marcas=5): " FLAGS; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"cd $(VPS_PATH) && ./$(BINARY) $$FLAGS"

## Roda migrate (up) localmente
migrate:
	@echo "⬆️  Aplicando migrations..."
	MONGO_URI=$(MONGO_URI) MONGO_DATABASE=$(MONGO_DATABASE) go run ./cmd/migrate -cmd=up

## Status das migrations localmente
migrate-status:
	MONGO_URI=$(MONGO_URI) MONGO_DATABASE=$(MONGO_DATABASE) go run ./cmd/migrate -cmd=status

## Reverte a última migration localmente
migrate-down:
	MONGO_URI=$(MONGO_URI) MONGO_DATABASE=$(MONGO_DATABASE) go run ./cmd/migrate -cmd=down

## Roda migrate (up) na VPS via SSH
migrate-vps:
	@read -sp "Senha SSH: " PASS; echo; \
	echo "⬆️  Aplicando migrations na VPS..."; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"cd $(VPS_PATH) && ./$(BINARY_MIGRATE) -cmd=up"

## Remove binários gerados localmente
clean:
	@rm -rf $(BUILD_DIR)
	@echo "🧹 Build limpo."
