package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var database *mongo.Database

func Connect(ctx context.Context) error {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return fmt.Errorf("variável MONGO_URI não definida — configure o .env")
	}

	dbName := os.Getenv("MONGO_DATABASE")
	if dbName == "" {
		return fmt.Errorf("variável MONGO_DATABASE não definida — configure o .env")
	}

	opts := options.Client().ApplyURI(uri)

	var err error
	client, err = mongo.Connect(ctx, opts)
	if err != nil {
		return err
	}

	// Ping para confirmar conexão
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = client.Ping(pingCtx, nil); err != nil {
		return err
	}

	database = client.Database(dbName)
	return nil
}

func Disconnect() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Erro ao desconectar do MongoDB: %v", err)
		}
	}
}

func GetCollection(name string) *mongo.Collection {
	return database.Collection(name)
}

func GetDatabase() *mongo.Database {
	return database
}
