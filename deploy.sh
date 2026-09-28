#!/bin/bash
set -e

# Carrega variáveis do .env se existir
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
else
  echo "⚠️ Arquivo .env não encontrado, certifique-se de configurar as variáveis de ambiente!"
fi

BINARY="fipe-scraper"
BINARY_MIGRATE="fipe-migrate"
BUILD_DIR="./build"

echo "🧪 Executando testes..."
go test ./...

echo "🔨 Compilando binários para Linux (amd64)..."
mkdir -p "$BUILD_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o "$BUILD_DIR/$BINARY" ./cmd/scraper
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o "$BUILD_DIR/$BINARY_MIGRATE" ./cmd/migrate
echo "✅ Binários gerados em $BUILD_DIR"

if [ -z "$VPS_USER" ] || [ -z "$VPS_HOST" ] || [ -z "$VPS_PATH" ]; then
  echo "⚠️ Variáveis VPS_USER, VPS_HOST ou VPS_PATH não configuradas. Encerrando apenas com o build."
  exit 0
fi

read -sp "🔑 Senha SSH: " PASS
echo

echo "🚀 Criando diretório remoto se necessário..."
sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no "$VPS_USER@$VPS_HOST" "mkdir -p $VPS_PATH"

echo "📤 Enviando binários e arquivos de configuração para VPS..."
sshpass -p "$PASS" scp -o StrictHostKeyChecking=no \
    "$BUILD_DIR/$BINARY" \
    "$BUILD_DIR/$BINARY_MIGRATE" \
    "$VPS_USER@$VPS_HOST:$VPS_PATH/"

if [ -f fipe-scraper.service ]; then
  sshpass -p "$PASS" scp -o StrictHostKeyChecking=no fipe-scraper.service "$VPS_USER@$VPS_HOST:$VPS_PATH/fipe-scraper.service"
fi

if [ -f .env ]; then
  sshpass -p "$PASS" scp -o StrictHostKeyChecking=no .env "$VPS_USER@$VPS_HOST:$VPS_PATH/.env"
fi

echo "✅ Deploy concluído! Binários e configurações atualizados em $VPS_PATH na VPS."
