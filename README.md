# 🚗 FIPE Scraper

Scraper robusto e de alto desempenho em Go que coleta preços de veículos da [Tabela FIPE](https://veiculos.fipe.org.br/) e salva no MongoDB.

---

## 🛠️ Stack & Tecnologias

- **Go 1.23** (com Context, Graceful Shutdown e suporte a sinais OS)
- **MongoDB** (driver oficial com índices e migrations)
- **Docker** & **Docker Compose** (Multi-stage build leve)
- **Systemd / Cron / Makefile** (Opções flexíveis de deploy em produção)

---

## 📋 Atalhos Rápidos (Makefile)

O projeto possui atalhos padronizados para facilitar o uso local e em produção:

```bash
make check-setup   # 🔍 Verifica se Docker, Docker Compose, Go e .env estão configurados
make setup         # ⚙️ Cria .env inicial e baixa dependências Go
make build         # 🔨 Compila os binários (fipe-scraper e fipe-migrate) para Linux
make up            # 🚀 Sobe a stack completa via Docker Compose (MongoDB + Migrate + Scraper)
make down          # 🛑 Para e remove os containers Docker
make test          # 🧪 Executa os testes unitários
make deploy        # 📤 Envia binários e configurações para a VPS por SSH
```

---

## 📁 Estrutura do Projeto

```
fipe-scraper/
├── cmd/
│   ├── scraper/main.go          # Entry point do scraper
│   └── migrate/main.go          # CLI de migrations do MongoDB
├── internal/
│   ├── db/mongo.go              # Conexão e gerenciamento MongoDB
│   ├── migrations/              # Definição e execução de migrations
│   ├── models/models.go         # Structs e mapeamento de dados
│   └── scraper/                 # Cliente de API da FIPE e retentativas
│       ├── fipe.go
│       └── fipe_test.go
├── Dockerfile                   # Build multi-stage seguro (Alpine)
├── docker-compose.yml           # Stack completa (MongoDB + Migrate + Scraper)
├── fipe-scraper.service         # Template systemd para produção
├── Makefile                     # Atalhos de build, test e deploy
├── deploy.sh                    # Script de deploy SSH
└── .env.example                 # Exemplo de variáveis de ambiente
```

---

## ⚙️ Variáveis de Ambiente

Crie um arquivo `.env` baseado no `.env.example` (ou rode `make setup`):

| Variável | Padrão | Descrição |
|---|---|---|
| `MONGO_URI` | `mongodb://root:example@mongodb:27017/?authSource=admin` | URI de conexão com MongoDB |
| `MONGO_DATABASE` | `fipe` | Nome do banco de dados |
| `REQUEST_DELAY_MS` | `1200` | Delay entre requisições na FIPE (evita rate limit) |
| `VPS_USER` | `root` | Usuário SSH para deploy remoto |
| `VPS_HOST` | `123.456.78.90` | IP ou Host da VPS em produção |
| `VPS_PATH` | `/opt/fipe-scraper` | Diretório da aplicação na VPS |

---

## 🚩 Flags CLI

O scraper aceita parâmetros para limitar a execução ou selecionar tabelas específicas:

| Flag | Padrão | Descrição |
|---|---|---|
| `-tabela` | `0` | Código da tabela de referência (`0` = mais recente) |
| `-marcas` | `0` | Limitar quantidade de marcas por tabela (`0` = todas) |
| `-tabelas` | `0` | Limitar quantidade de tabelas processadas (`0` = todas) |

---

## 🚀 Como Executar

### 1. Com Docker Compose (Recomendado para Desenvolvimento e Produção Isolada)

Configuração rápida em 2 comandos:

```bash
# 1. Configurar ambiente
make setup

# 2. Subir a stack inteira (MongoDB + Migrate + Scraper)
make up
```

Para ver os logs do scraper:

```bash
docker compose logs -f scraper
```

Para parar os containers:

```bash
make down
```

---

## 🏭 Deploy em Máquina em Produção (VPS / Servidor Linux)

Para rodar diretamente na máquina sem Docker (ou integrando ao MongoDB existente), você tem duas opções principais em produção:

### Opção A: Usando Systemd Service (Recomendado para Execução Contínua ou Reinício Automático)

1. Compilar os binários e enviar para o servidor:
```bash
make build-deploy
# ou via script: ./deploy.sh
```

2. Na VPS, configure o serviço systemd:
```bash
# Copie o arquivo de serviço para o systemd
cp /opt/fipe-scraper/fipe-scraper.service /etc/systemd/system/

# Recarregue o daemon do systemd
systemctl daemon-reload

# Habilite e inicie o serviço
systemctl enable fipe-scraper
systemctl start fipe-scraper

# Acompanhe os logs
journalctl -u fipe-scraper -f
```

---

### Opção B: Agendamento Automático via Cron (Para rodar todo mês/dia)

Caso queira rodar o scraper apenas no primeiro dia de cada mês:

```bash
crontab -e
```

Adicione a linha (executa todo dia 1 às 08:00):

```bash
# Via Docker Compose:
0 8 1 * * cd /opt/fipe-scraper && make up

# Ou via binário direto:
0 8 1 * * cd /opt/fipe-scraper && ./fipe-scraper >> /opt/fipe-scraper/cron.log 2>&1
```

---

## 🧪 Testes Unitários e Compilação Local

```bash
# Verificar ambiente
make check-setup

# Executar todos os testes
make test

# Compilar binários locais para Linux
make build

# Limpar builds antigos
make clean
```
