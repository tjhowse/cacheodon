package main

import (
	"context"
	"fmt"
	"log"
	"os"
)

func main() {
	var err error
	ctx := context.Background()
	g, _ := NewGeocachingAPI(ctx)
	if err := g.Auth(os.Getenv("GEOCACHING_CLIENT_ID"), os.Getenv("GEOCACHING_CLIENT_SECRET")); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	log.Println("Authenticated")
	var searchResults []Geocache
	if searchResults, err = g.Search(-27.46794, 153.02809); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Println("Found", len(searchResults), "geocaches")
	// m := NewMastodon()
	// if err := m.PostStatus(fmt.Sprintf("I know about %d geocaches around Brisbane! I'm being taught how to tell you about them in a useful way. Stand by!", len(searchResults))); err != nil {
	// 	log.Fatal(err)
	// }
}
