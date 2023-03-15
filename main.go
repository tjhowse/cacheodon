package main

import (
	"flag"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// This truncates a string to the given maximum length and returns
// the result. If truncation was necessary, it adds an elipsis to
// the end of the string.
func truncate(s string, max int) string {
	if len(s) >= max {
		return s[:max-1] + "…"
	}
	return s
}
func printType(i interface{}) {
	switch v := i.(type) {
	case int:
		log.Printf("type of %v is %v\n", i, v)
		// type of 21 is int
	case string:
		log.Printf("type of %v is %v\n", i, v)
		// type of hello is string
	default:
		log.Printf("type of %v is %v\n", i, v)
		// type of true is bool
	}
}
func main() {
	var err error

	verbose := flag.Bool("v", false, "Verbose logging")

	flag.Parse()
	if *verbose {
		// Set the log level to debug
		log.SetLevel(log.DebugLevel)
	}
	// Set the log format to include a leading timestamp in ISO8601 format
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	config, err := NewDatastore("config.toml")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	var g *Geocaching
	if g, err = NewGeocaching(config.Store); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer g.Close()
	var m *Mastodon
	for {
		if posts, err := g.Update(); err == nil {
			for _, post := range posts {
				if m == nil {
					m, err = NewMastodon()
					if err != nil {
						log.Println(err)
					}
				}
				if err := m.PostStatus(post); err != nil {
					log.Println(err)
					m = nil
				} else {
					log.Println("Posted to Mastodon: " + post)
				}
				// Wait a random number of seconds between 3 and 8
				time.Sleep(time.Duration(rand.Intn(5)+3) * time.Second)
			}
		} else {
			log.Println(err)
		}
		// Wait a random number of minutes between 3 and 8
		time.Sleep(time.Duration(rand.Intn(5*60)+3*60) * time.Second)
	}

}
