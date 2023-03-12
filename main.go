package main

import (
	"flag"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dustin/go-humanize"
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

	cacheDB, err := NewFinderDB("cacheodon.sqlite3")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer cacheDB.Close()

	var g *GeocachingAPI
	if g, err = NewGeocachingAPI(config.Store.Configuration); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	if err = g.Auth(os.Getenv("GEOCACHING_CLIENT_ID"), os.Getenv("GEOCACHING_CLIENT_SECRET")); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	log.Println("Authenticated")
	var searchResults []Geocache
	m, err := NewMastodon()
	if err != nil {
		log.Println(err)
	}

	for {
		// This should use cacheDB.GetLastPostedFoundTime(time.Now()) instead of
		// config.Store.State.LastPostedFoundTime but it needs testing.
		cacheDB.GetLastPostedFoundTime(time.Now())
		if searchResults, err = g.SearchSince(
			config.Store.SearchTerms,
			config.Store.State.LastPostedFoundTime); err != nil {

			log.Println(err)
			time.Sleep(1 * time.Minute)
			continue
		}
		for _, gc := range searchResults {
			logs, err := g.GetLogs(&gc)
			if err != nil || len(logs) == 0 {
				log.Println(err)
				continue
			}

			cacheDB.AddLog(&logs[0], &gc)

			log.Debug("This log type is \"" + logs[0].LogType + "\"")
			// Print whatever the images are if the log has one:
			if len(logs[0].Images) > 0 {
				for _, image := range logs[0].Images {
					printType(image)
				}
			}
			// TODO Consider trying to grab a photo from the log and attach it to the post

			message := ""
			message += "In " + config.Store.SearchTerms.AreaName + ", \"" + logs[0].UserName + "\""
			message += " just found the \"" + gc.Name + "\" geocache! https://www.geocaching.com" + gc.DetailsURL
			if findCount := cacheDB.FindsSinceMidnight(logs[0].UserName); findCount > 1 {
				message += " That's their " + humanize.Ordinal(findCount) + " find today!"
			}
			message += " They wrote: \"" + logs[0].LogText + "\""

			if m == nil {
				m, err = NewMastodon()
				if err != nil {
					log.Println(err)
				}
			}
			geocachingHashtagString := " #geocaching"
			message = truncate(message, 500-len(geocachingHashtagString))
			message += geocachingHashtagString
			if err := m.PostStatus(message); err != nil {
				log.Println(err)
				m = nil
			} else {
				log.Println("Posted to Mastodon: " + message)
				cacheDB.SetLastPostedFoundTime(gc.LastFoundTime)
				config.Store.State.LastPostedFoundTime = gc.LastFoundTime
				if err := config.Save(); err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
			}
			// Wait a random number of seconds between 3 and 8
			time.Sleep(time.Duration(rand.Intn(5)+3) * time.Second)
		}
		// Wait a random number of minutes between 3 and 8
		time.Sleep(time.Duration(rand.Intn(5*60)+3*60) * time.Second)
	}

}
