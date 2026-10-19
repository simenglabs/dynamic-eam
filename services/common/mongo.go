package common

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Mongo connects to MONGO_URI (default mongodb://localhost:27017), database MONGO_DB (default "eam").
func Mongo() *mongo.Database {
	uri := Env("MONGO_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetTimeout(10 * time.Second).
		// Nested documents decode to bson.M (map) so they can be JSON-encoded directly.
		SetBSONOptions(&options.BSONOptions{DefaultDocumentM: true}))
	if err != nil {
		log.Fatal("mongo: ", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("mongo ping: ", err)
	}
	return client.Database(Env("MONGO_DB", "eam"))
}

func Env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
