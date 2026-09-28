package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/marcuspavao/fipe-scraper/internal/db"
	"github.com/marcuspavao/fipe-scraper/internal/models"
	"github.com/marcuspavao/fipe-scraper/internal/scraper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	tabelaFlag := flag.Int("tabela", 0, "Código da tabela (0 = mais recente)")
	limMarcas := flag.Int("marcas", 0, "Limitar marcas (0 = todas)")
	limTabelas := flag.Int("tabelas", 0, "Limitar tabelas de referência (0 = todas)")
	flag.Parse()

	log.Println("🚗 FIPE Scraper iniciando...")

	// Contexto global com cancelamento por sinais de interrupção (SIGINT, SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connCtx, cancelConn := context.WithTimeout(ctx, 10*time.Second)
	defer cancelConn()

	if err := db.Connect(connCtx); err != nil {
		log.Fatalf("❌ MongoDB: %v", err)
	}
	defer db.Disconnect()
	log.Println("✅ MongoDB conectado")

	// ─── 1. Tabelas de referência ───────────────────────────────────────────
	log.Println("📅 Buscando tabelas de referência...")
	tabelas, err := scraper.BuscarTabelasWithContext(ctx)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			log.Println("🛑 Operação cancelada pelo usuário")
			os.Exit(0)
		}
		log.Fatalf("❌ %v", err)
	}

	// Salva TODAS as tabelas na collection TabelaReferencia
	colTabelas := db.GetCollection("TabelaReferencia")
	for _, t := range tabelas {
		if ctx.Err() != nil {
			log.Println("🛑 Encerrando antes de salvar tabelas...")
			os.Exit(0)
		}
		filter := bson.M{"codigo": t.Codigo}
		update := bson.M{"$set": t}
		_, _ = colTabelas.UpdateOne(
			ctx, filter, update,
			options.Update().SetUpsert(true),
		)
	}
	log.Printf("   💾 %d tabelas salvas em TabelaReferencia", len(tabelas))

	inicioIdx := 0
	if *tabelaFlag != 0 {
		for i, t := range tabelas {
			if t.Codigo == *tabelaFlag {
				inicioIdx = i
				break
			}
		}
	}

	fimIdx := len(tabelas)
	if *limTabelas > 0 && inicioIdx+*limTabelas < fimIdx {
		fimIdx = inicioIdx + *limTabelas
	}

	log.Printf("   🗓️  Processando %d tabelas a partir de [%s]", fimIdx-inicioIdx, tabelas[inicioIdx].Mes)

	colVeiculos := db.GetCollection("Veiculos")

	// ─── Itera tabelas regressivamente ──────────────────────────────────────
	for tIdx := inicioIdx; tIdx < fimIdx; tIdx++ {
		if ctx.Err() != nil {
			log.Println("🛑 Sinal de encerramento recebido. Parando processamento de tabelas...")
			break
		}

		tabela := tabelas[tIdx]
		log.Printf("\n════════════════════════════════════════════════════════")
		log.Printf("📅 [%d/%d] Tabela: [%d] %s", tIdx-inicioIdx+1, fimIdx-inicioIdx, tabela.Codigo, tabela.Mes)
		log.Printf("════════════════════════════════════════════════════════")

		// ─── 2. Marcas ──────────────────────────────────────────────────────
		marcas, err := scraper.BuscarMarcasWithContext(ctx, tabela.Codigo)
		if err != nil {
			log.Printf("❌ Erro ao buscar marcas para [%s]: %v", tabela.Mes, err)
			continue
		}
		if *limMarcas > 0 && len(marcas) > *limMarcas {
			marcas = marcas[:*limMarcas]
		}
		log.Printf("   🏭 %d marcas encontradas", len(marcas))

		salvos, erros := 0, 0

		// ─── 3. Itera marcas ────────────────────────────────────────────────
		for i, marca := range marcas {
			if ctx.Err() != nil {
				log.Println("🛑 Sinal de encerramento recebido. Parando processamento de marcas...")
				break
			}

			codigoMarca, _ := strconv.Atoi(marca.Value)
			log.Printf("[%d/%d] 🏭 %s  (tabela %s)", i+1, len(marcas), marca.Label, strings.TrimSpace(tabela.Mes))

			var modelosResp *models.FIPEModelosResp
			for {
				if ctx.Err() != nil {
					break
				}
				modelosResp, err = scraper.BuscarModelosWithContext(ctx, tabela.Codigo, codigoMarca)
				if err == nil {
					break
				}
				if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "429") {
					log.Printf("   ⚠️ BLOQUEIO 403/429 em Modelos para [%s]. Aguardando 1 minuto...", marca.Label)
					if sleepWithContext(ctx, 1*time.Minute) != nil {
						break
					}
					continue
				}
				break
			}

			if ctx.Err() != nil {
				log.Println("🛑 Cancelado durante busca de modelos.")
				break
			}

			if err != nil {
				log.Printf("   ❌ FALHA DEFINITIVA em Modelos para [%s]: %v", marca.Label, err)
				erros++
				continue
			}

			// ─── Carrega documento existente para evitar duplicidade ─────────
			var existingDoc models.VeiculoDoc
			filter := bson.M{"monthYearId": tabela.Codigo, "brandCode": codigoMarca}
			err = colVeiculos.FindOne(ctx, filter).Decode(&existingDoc)
			hasExistingDoc := err == nil

			modelDocsMap := make(map[int]models.ModelData)
			if hasExistingDoc {
				for _, m := range existingDoc.Models {
					modelDocsMap[m.ModelCode] = m
				}
			}

			var modelDocs []models.ModelData

			// ─── 4. Itera modelos ────────────────────────────────────────────
			for _, modelo := range modelosResp.Modelos {
				if ctx.Err() != nil {
					log.Println("🛑 Cancelado durante iteração de modelos.")
					break
				}

				// Se o modelo já existe no banco com preços, pula
				if hasExistingDoc {
					if mExistente, ok := modelDocsMap[modelo.Value]; ok && len(mExistente.Years) > 0 {
						todasComPreco := true
						for _, y := range mExistente.Years {
							if y.Price == "" {
								todasComPreco = false
								break
							}
						}
						if todasComPreco {
							log.Printf("      ⏩ %s já existe no banco (%d anos com preços)", modelo.Label, len(mExistente.Years))
							modelDocs = append(modelDocs, mExistente)
							continue
						}
					}
				}

				var years []models.YearData

				// ─── 5. Busca anos para o modelo ─────────────────────────────
				var anos []models.FIPEAnoModelo
				for {
					if ctx.Err() != nil {
						break
					}
					anos, err = scraper.BuscarAnosWithContext(ctx, tabela.Codigo, codigoMarca, modelo.Value)
					if err == nil {
						break
					}
					if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "429") {
						log.Printf("      ⚠️ BLOQUEIO 403/429 em Anos para [%s]. Aguardando 1 minuto...", modelo.Label)
						if sleepWithContext(ctx, 1*time.Minute) != nil {
							break
						}
						continue
					}
					break
				}

				if ctx.Err() != nil {
					break
				}

				if err != nil {
					log.Printf("      ❌ FALHA DEFINITIVA em Anos para [%s]: %v", modelo.Label, err)
					erros++
					continue
				}

				// ─── 6. Itera anos ───────────────────────────────────────────
				for _, ano := range anos {
					if ctx.Err() != nil {
						break
					}

					anoInt, combInt, err := scraper.ParseAnoModelo(ano.Value)
					if err != nil {
						continue
					}

					var valor *models.FIPEValorResp
					for {
						if ctx.Err() != nil {
							break
						}
						if sleepWithContext(ctx, scraper.GetRequestDelay()) != nil {
							break
						}
						valor, err = scraper.BuscarValorWithContext(ctx, tabela.Codigo, codigoMarca, modelo.Value, anoInt, combInt)
						if err == nil {
							break
						}
						if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "429") {
							log.Printf("      ⚠️ BLOQUEIO 403/429 em Valor para [%s] (%s). Aguardando 1 minuto...", modelo.Label, ano.Label)
							if sleepWithContext(ctx, 1*time.Minute) != nil {
								break
							}
							continue
						}
						break
					}

					if ctx.Err() != nil {
						break
					}

					if err != nil {
						log.Printf("      ❌ FALHA DEFINITIVA em Valor para [%s] (%s): %v", modelo.Label, ano.Label, err)
						erros++
						continue
					}

					years = append(years, models.YearData{
						YearCode:       ano.Value,
						Year:           anoInt,
						Price:          valor.Valor,
						MonthReference: valor.MesReferencia,
					})
				}

				if len(years) > 0 {
					modelDocs = append(modelDocs, models.ModelData{
						ModelCode: modelo.Value,
						ModelName: modelo.Label,
						Years:     years,
					})
					log.Printf("      💾 [%s] coletado (%d anos)", modelo.Label, len(years))
				}

				if sleepWithContext(ctx, scraper.GetRequestDelay()) != nil {
					break
				}
			}

			if ctx.Err() != nil {
				log.Println("🛑 Cancelado antes de persistir marca.")
				break
			}

			// ─── 7. Upsert da marca na collection Veiculos ───────────────────
			doc := models.VeiculoDoc{
				MonthYearID: tabela.Codigo,
				BrandCode:   codigoMarca,
				BrandName:   marca.Label,
				Models:      modelDocs,
			}
			filter = bson.M{"monthYearId": tabela.Codigo, "brandCode": codigoMarca}
			update := bson.M{"$set": doc}
			_, err = colVeiculos.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
			if err != nil {
				log.Printf("   ⚠️  Erro ao salvar %s: %v", marca.Label, err)
				erros++
			} else {
				salvos++
				log.Printf("   ✅ %s salva (%d modelos)", marca.Label, len(modelDocs))
			}

			if sleepWithContext(ctx, scraper.GetRequestDelay()) != nil {
				break
			}
		}

		log.Printf("\n   📊 Tabela [%s] finalizada — ✅ %d marcas salvas  ⚠️ %d erros", strings.TrimSpace(tabela.Mes), salvos, erros)
	}

	if ctx.Err() != nil {
		fmt.Printf("\n════════════════════════════════════\n")
		fmt.Printf("🛑 Scraping interrompido pelo usuário/sistema.\n")
		fmt.Printf("════════════════════════════════════\n")
		os.Exit(130)
	}

	fmt.Printf("\n════════════════════════════════════\n")
	fmt.Printf("✅ Scraping completo de todas as tabelas!\n")
	fmt.Printf("════════════════════════════════════\n")

	os.Exit(0)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
