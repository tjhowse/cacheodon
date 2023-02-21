package main

import (
	"context"
	"log"
	"os"
)

func main() {
	// m := NewMastodon()
	// if err := m.PostStatus("Hello, worlds!"); err != nil {
	// 	log.Fatal(err)
	// }
	ctx := context.Background()
	g := NewGeocachingAPI(ctx)
	if err := g.Auth(os.Getenv("GEOCACHING_CLIENT_ID"), os.Getenv("GEOCACHING_CLIENT_SECRET")); err != nil {
		log.Fatal(err)
	}
	if err, _ := g.Search(-27.46794, 153.02809); err != nil {
		log.Fatal(err)
	}
}
