# Stage 1: Build binaries
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Instala certificados de CA
RUN apk add --no-cache ca-certificates tzdata

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copia código fonte
COPY . .

# Compila os binários de forma estática
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/build/fipe-scraper ./cmd/scraper
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/build/fipe-migrate ./cmd/migrate

# Stage 2: Runtime leve e seguro
FROM alpine:3.20

# Instala tzdata e ca-certificates
RUN apk add --no-cache ca-certificates tzdata

# Cria usuário não-root por segurança
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copia os binários do builder
COPY --from=builder /app/build/fipe-scraper /app/fipe-scraper
COPY --from=builder /app/build/fipe-migrate /app/fipe-migrate

# Ajusta permissões
RUN chown -R appuser:appgroup /app

USER appuser

ENTRYPOINT ["/app/fipe-scraper"]
