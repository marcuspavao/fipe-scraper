-include .env
export

BINARY         := fipe-scraper
BINARY_MIGRATE := fipe-migrate
BUILD_DIR      := ./build

.PHONY: build build-migrate deploy build-deploy run start logs stop migrate migrate-status migrate-down migrate-vps clean

## Compila o scraper para Linux (AMD64)
build:
	@echo "🔨 Compilando scraper para Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY) ./cmd/scraper
	@echo "✅ $(BUILD_DIR)/$(BINARY) gerado"

## Compila o migrate para Linux (AMD64)
build-migrate:
	@echo "🔨 Compilando migrate para Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_MIGRATE) ./cmd/migrate
	@echo "✅ $(BUILD_DIR)/$(BINARY_MIGRATE) gerado"

## Faz o deploy do scraper + migrate para a VPS (pede senha)
deploy:
	@echo "🚀 Enviando binários para $(VPS_USER)@$(VPS_HOST):$(VPS_PATH)..."
	@read -sp "Senha SSH: " PASS; echo; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) "mkdir -p $(VPS_PATH)"; \
	sshpass -p "$$PASS" scp -o StrictHostKeyChecking=no $(BUILD_DIR)/$(BINARY) $(VPS_USER)@$(VPS_HOST):$(VPS_PATH)/$(BINARY); \
	sshpass -p "$$PASS" scp -o StrictHostKeyChecking=no $(BUILD_DIR)/$(BINARY_MIGRATE) $(VPS_USER)@$(VPS_HOST):$(VPS_PATH)/$(BINARY_MIGRATE)
	@echo "✅ Deploy concluído!"

## Build (scraper + migrate) + Deploy em um único comando
build-deploy: build build-migrate deploy

## Roda o scraper em background na VPS via nohup (pede senha e flags)
start:
	@read -sp "Senha SSH: " PASS; echo; \
	read -p "Flags (ex: -tabelas=10 -marcas=5): " FLAGS; \
	echo "🚀 Iniciando scraper na VPS em background..."; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"cd $(VPS_PATH) && nohup ./$(BINARY) $$FLAGS > scraper.log 2>&1 & echo \"PID: \$$!\""
	@echo "✅ Scraper rodando! Use 'make logs' para acompanhar."

## Acompanha o log do scraper em tempo real na VPS (pede senha)
logs:
	@read -sp "Senha SSH: " PASS; echo; \
	echo "📋 Acompanhando log... (Ctrl+C para sair)"; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"tail -f $(VPS_PATH)/scraper.log"

## Para o scraper em execução na VPS (pede senha)
stop:
	@read -sp "Senha SSH: " PASS; echo; \
	echo "🛑 Parando scraper na VPS..."; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"pkill -f $(BINARY) && echo '✅ Parado.' || echo '⚠️  Nenhum processo encontrado.'"

## Roda o scraper interativamente na VPS (trava o terminal até terminar)
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

## Roda migrate (up) na VPS via SSH (pede senha)
migrate-vps:
	@read -sp "Senha SSH: " PASS; echo; \
	echo "⬆️  Aplicando migrations na VPS..."; \
	sshpass -p "$$PASS" ssh -o StrictHostKeyChecking=no $(VPS_USER)@$(VPS_HOST) \
		"cd $(VPS_PATH) && ./$(BINARY_MIGRATE) -cmd=up"

## Remove binários gerados localmente
clean:
	@rm -rf $(BUILD_DIR)
	@echo "🧹 Build limpo."
