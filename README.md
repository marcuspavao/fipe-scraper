# 🚗 FIPE Scraper

Scraper em Go que coleta preços de veículos da [Tabela FIPE](https://veiculos.fipe.org.br/) e salva no MongoDB.

## Stack

- **Go 1.23**
- **MongoDB** (driver oficial)
- **Docker** + **Docker Compose**

## Estrutura

```
fipe-scraper/
├── cmd/scraper/main.go          # Entry point (CLI com flags)
├── internal/
│   ├── db/mongo.go              # Conexão MongoDB
│   ├── models/models.go         # Structs de dados
│   └── scraper/fipe.go          # Cliente da API FIPE
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## Como usar

### Com Docker (recomendado)

```bash
# Scraping completo (todas as marcas da tabela mais recente)
docker-compose up

# Com limites (para teste)
docker-compose run scraper ./fipe-scraper -marcas=5 -modelos=3

# Tabela específica
docker-compose run scraper ./fipe-scraper -tabela=319
```

### Local (requer Go 1.23+ e MongoDB)

```bash
cp .env.example .env
go mod tidy

# Todas as marcas, tabela mais recente
go run ./cmd/scraper

# Limitado para teste rápido
go run ./cmd/scraper -marcas=3 -modelos=2
```

## Flags disponíveis

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-tabela` | `0` | Código da tabela de referência (`0` = mais recente) |
| `-marcas` | `0` | Limitar quantidade de marcas (`0` = todas) |
| `-modelos` | `0` | Limitar modelos por marca (`0` = todos) |

## Dados salvos no MongoDB

Collection: `veiculos`

```json
{
  "tabela_codigo": 319,
  "tabela_mes": "março/2026",
  "marca": "Fiat",
  "modelo": "Argo Drive 1.0",
  "ano_modelo": 2022,
  "codigo_fipe": "001297-0",
  "preco": 68590.00,
  "preco_str": "R$ 68.590,00",
  "combustivel": "Gasolina",
  "mes_referencia": "março de 2026",
  "data_consulta": "2026-03-09T22:00:00Z"
}
```

## Como agendar (cron no servidor)

```bash
# Rodar todo dia 1 às 08h
0 8 1 * * cd /home/marcus_dev/projects/fipe-scraper && docker-compose up --build
```
# fipe-scraper
