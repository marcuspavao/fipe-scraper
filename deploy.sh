#!/bin/bash
set -e

# Carrega variáveis do .env
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
else
  echo "❌ Arquivo .env não encontrado!"
  exit 1
fi

BINARY="fipe-scraper"
BUILD_DIR="./build"

echo "🔨 Compilando para Linux (amd64)..."
mkdir -p "$BUILD_DIR"
GOOS=linux GOARCH=amd64 go build -o "$BUILD_DIR/$BINARY" ./cmd/scraper
echo "✅ Binário gerado em $BUILD_DIR/$BINARY"

read -sp "🔑 Senha SSH: " PASS
echo

echo "🚀 Criando diretório remoto se necessário..."
sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no "$VPS_USER@$VPS_HOST" "mkdir -p $VPS_PATH"

echo "📤 Enviando binário para VPS..."
sshpass -p "$PASS" scp -o StrictHostKeyChecking=no \
    "$BUILD_DIR/$BINARY" \
    "$VPS_USER@$VPS_HOST:$VPS_PATH/$BINARY"

echo "✅ Deploy concluído! Binário disponível em $VPS_PATH/$BINARY na VPS."
