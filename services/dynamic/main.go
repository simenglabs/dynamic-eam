// Service dynamic: menu dinamis ala Strapi (schema, layout, workflow, entri, media). main = composition root.
package main

import (
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"eam/services/common"
	"eam/services/dynamic/internal/app"
	"eam/services/dynamic/internal/domain/content"
	dynhttp "eam/services/dynamic/internal/http"
	"eam/services/dynamic/internal/infra/disk"
	"eam/services/dynamic/internal/infra/elastic"
	"eam/services/dynamic/internal/infra/mongo"
	"eam/services/dynamic/internal/seed"
	"github.com/gin-gonic/gin"
)

func main() {
	db := common.Mongo()
	store := mongo.New(db)
	if err := store.EnsureIndexes(context.Background()); err != nil {
		log.Fatal(err)
	}
	files, err := disk.New(common.Env("MEDIA_DIR", "./data/media"))
	if err != nil {
		log.Fatal(err)
	}
	index := elastic.New(common.Env("ES_URL", "http://localhost:9200"), os.Getenv("ES_API_KEY"))
	if n, err := strconv.Atoi(common.Env("ES_REPLICAS", "1")); err == nil {
		index.Replicas = n
	}
	d := app.Deps{
		Schemas: store, Entries: store, MediaRepo: store.Media(), Files: files,
		Index: index,
		Now:   func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) },
		NewID: rand.Text,
	}

	// Menu & data bawaan (Master, Assets) dibuat bila belum ada; yang sudah ada tidak ditimpa.
	system := content.Actor{ID: "system", Name: "System", Super: true}
	for _, name := range seed.Order {
		comps, cts, rows, err := dynhttp.DecodeSeed(seed.Files[name])
		if err != nil {
			log.Fatalf("seed %s: %v", name, err)
		}
		created, err := app.NewSchemas(d).Ensure(context.Background(), comps, cts)
		if err != nil {
			log.Printf("seed %s: %v", name, err) // mis. index ES gagal; schema tetap tersimpan
		}
		for _, r := range rows {
			done, err := app.NewEntries(d).EnsureEntries(context.Background(), r.UID, r.Key, r.Rows, system)
			if err != nil {
				log.Printf("seed %s data %s: %v", name, r.UID, err)
			}
			created = append(created, done...)
		}
		if len(created) > 0 {
			log.Printf("seed %s: dibuat %v", name, created)
		}
	}

	if err := app.NewSchemas(d).SyncConstraints(context.Background()); err != nil {
		log.Print("sync constraint: ", err) // mis. data lama duplikat; service tetap jalan
	}

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	dynhttp.Register(r.Group("/api/dynamic"), app.NewSchemas(d), app.NewEntries(d), app.NewMedia(d), common.NewAuthz(db))
	r.Run(":" + common.Env("PORT", "8088"))
}
