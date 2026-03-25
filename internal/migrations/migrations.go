package migrations

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Migration representa uma migration individual
type Migration struct {
	ID          string
	Description string
	Up          func(ctx context.Context, db *mongo.Database) error
	Down        func(ctx context.Context, db *mongo.Database) error
}

// migrationRecord é salvo na collection "_migrations" do banco
type migrationRecord struct {
	ID          string    `bson:"_id"`
	Description string    `bson:"description"`
	AppliedAt   time.Time `bson:"appliedAt"`
}

// All retorna todas as migrations em ordem de execução.
// Adicione novas migrations SEMPRE ao final da lista.
func All() []Migration {
	return []Migration{
		{
			ID:          "001_indexes_veiculos",
			Description: "Cria índice único em Veiculos (monthYearId + brandCode)",
			Up: func(ctx context.Context, db *mongo.Database) error {
				_, err := db.Collection("Veiculos").Indexes().CreateOne(ctx, mongo.IndexModel{
					Keys:    bson.D{{Key: "monthYearId", Value: 1}, {Key: "brandCode", Value: 1}},
					Options: options.Index().SetUnique(true).SetName("idx_veiculos_unique"),
				})
				return err
			},
			Down: func(ctx context.Context, db *mongo.Database) error {
				_, err := db.Collection("Veiculos").Indexes().DropOne(ctx, "idx_veiculos_unique")
				return err
			},
		},
		{
			ID:          "002_indexes_tabela_referencia",
			Description: "Cria índice único em TabelaReferencia (codigo)",
			Up: func(ctx context.Context, db *mongo.Database) error {
				_, err := db.Collection("TabelaReferencia").Indexes().CreateOne(ctx, mongo.IndexModel{
					Keys:    bson.D{{Key: "codigo", Value: 1}},
					Options: options.Index().SetUnique(true).SetName("idx_tabela_codigo_unique"),
				})
				return err
			},
			Down: func(ctx context.Context, db *mongo.Database) error {
				_, err := db.Collection("TabelaReferencia").Indexes().DropOne(ctx, "idx_tabela_codigo_unique")
				return err
			},
		},
		{
			ID:          "003_index_veiculos_brandname",
			Description: "Cria índice em Veiculos (brandName) para busca por marca",
			Up: func(ctx context.Context, db *mongo.Database) error {
				_, err := db.Collection("Veiculos").Indexes().CreateOne(ctx, mongo.IndexModel{
					Keys:    bson.D{{Key: "brandName", Value: 1}},
					Options: options.Index().SetName("idx_veiculos_brandname"),
				})
				return err
			},
			Down: func(ctx context.Context, db *mongo.Database) error {
				_, err := db.Collection("Veiculos").Indexes().DropOne(ctx, "idx_veiculos_brandname")
				return err
			},
		},
		// ────────────────────────────────────────────────────────────────
		// Adicione novas migrations abaixo, SEMPRE ao final da lista.
		// Exemplo:
		// {
		//   ID:          "004_minha_migration",
		//   Description: "Descrição do que faz",
		//   Up: func(ctx context.Context, db *mongo.Database) error { ... },
		//   Down: func(ctx context.Context, db *mongo.Database) error { ... },
		// },
		// ────────────────────────────────────────────────────────────────
	}
}

// Run aplica todas as migrations pendentes
func Run(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("_migrations")

	applied, err := getApplied(ctx, col)
	if err != nil {
		return fmt.Errorf("ler migrations aplicadas: %w", err)
	}

	pending := 0
	for _, m := range All() {
		if applied[m.ID] {
			log.Printf("   ⏩ [%s] %s — já aplicada", m.ID, m.Description)
			continue
		}

		log.Printf("   ⬆️  [%s] %s...", m.ID, m.Description)
		if err := m.Up(ctx, db); err != nil {
			return fmt.Errorf("migration %s falhou: %w", m.ID, err)
		}

		_, err = col.InsertOne(ctx, migrationRecord{
			ID:          m.ID,
			Description: m.Description,
			AppliedAt:   time.Now(),
		})
		if err != nil {
			return fmt.Errorf("registrar migration %s: %w", m.ID, err)
		}

		log.Printf("   ✅ [%s] aplicada com sucesso", m.ID)
		pending++
	}

	if pending == 0 {
		log.Println("   ✅ Banco já está atualizado, nenhuma migration pendente.")
	} else {
		log.Printf("   🎉 %d migration(s) aplicada(s).", pending)
	}
	return nil
}

// Rollback reverte a última migration aplicada
func Rollback(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("_migrations")

	// Busca a última migration aplicada
	opts := options.FindOne().SetSort(bson.D{{Key: "appliedAt", Value: -1}})
	var last migrationRecord
	if err := col.FindOne(ctx, bson.M{}, opts).Decode(&last); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("   ℹ️  Nenhuma migration para reverter.")
			return nil
		}
		return fmt.Errorf("buscar última migration: %w", err)
	}

	// Encontra a migration correspondente
	var found *Migration
	for _, m := range All() {
		if m.ID == last.ID {
			m := m
			found = &m
			break
		}
	}
	if found == nil {
		return fmt.Errorf("migration %s não encontrada no código", last.ID)
	}

	log.Printf("   ⬇️  Revertendo [%s] %s...", found.ID, found.Description)
	if err := found.Down(ctx, db); err != nil {
		return fmt.Errorf("rollback %s falhou: %w", found.ID, err)
	}

	if _, err := col.DeleteOne(ctx, bson.M{"_id": last.ID}); err != nil {
		return fmt.Errorf("remover registro de migration: %w", err)
	}

	log.Printf("   ✅ [%s] revertida.", found.ID)
	return nil
}

// Status mostra quais migrations foram aplicadas e quais estão pendentes
func Status(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("_migrations")
	applied, err := getApplied(ctx, col)
	if err != nil {
		return err
	}

	fmt.Println("\n════════════════════════════════════════════")
	fmt.Println("  STATUS DAS MIGRATIONS")
	fmt.Println("════════════════════════════════════════════")
	for _, m := range All() {
		if applied[m.ID] {
			fmt.Printf("  ✅ [%s] %s\n", m.ID, m.Description)
		} else {
			fmt.Printf("  ⏳ [%s] %s  ← PENDENTE\n", m.ID, m.Description)
		}
	}
	fmt.Println("════════════════════════════════════════════")
	return nil
}

func getApplied(ctx context.Context, col *mongo.Collection) (map[string]bool, error) {
	cursor, err := col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	applied := make(map[string]bool)
	for cursor.Next(ctx) {
		var r migrationRecord
		if err := cursor.Decode(&r); err == nil {
			applied[r.ID] = true
		}
	}
	return applied, cursor.Err()
}
