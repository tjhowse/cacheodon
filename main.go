package main

import (
	"context"
	"log"
	"os"
	"time"
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
	filterTime, _ := time.Parse(time.RFC3339[:19], "2023-02-15T00:00:00")
	if searchResults, err = g.SearchSince(-27.46794, 153.02809, filterTime); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Println("Found", len(searchResults), "geocaches")
	log.Println("First one is", searchResults[0])
	log.Println("Last one is", searchResults[len(searchResults)-1])
	// m := NewMastodon()
	// if err := m.PostStatus(fmt.Sprintf("I know about %d geocaches around Brisbane! I'm being taught how to tell you about them in a useful way. Stand by!", len(searchResults))); err != nil {
	// 	log.Fatal(err)
	// }
}
