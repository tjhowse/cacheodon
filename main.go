package main

import (
	"log"
	"os"
	"time"

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

func main() {
	var err error

	config, err := NewDatastore("config.toml")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	findDB, err := NewFinderDB("finds.sqlite3")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer findDB.Close()

	g, _ := NewGeocachingAPI("https://www.geocaching.com")
	if err := g.Auth(os.Getenv("GEOCACHING_CLIENT_ID"), os.Getenv("GEOCACHING_CLIENT_SECRET")); err != nil {
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
		if searchResults, err = g.SearchSince(
			float64(config.Store.SearchTerms.Latitude),
			float64(config.Store.SearchTerms.Longitude),
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

			findDB.AddFind(logs[0].UserName, gc.LastFoundTime, gc.Code, logs[0].LogText)

			message := ""
			message += "In Brisbane, \"" + logs[0].UserName + "\""
			message += " just found the \"" + gc.Name + "\" geocache! https://www.geocaching.com" + gc.DetailsURL
			if findCount := findDB.FindsSinceMidnight(logs[0].UserName); findCount > 1 {
				message += " That's their " + humanize.Ordinal(findCount) + " find today!"
			}
			message += " They wrote: \"" + logs[0].LogText + "\""

			if m == nil {
				m, err = NewMastodon()
				if err != nil {
					log.Println(err)
				}
			}
			message = truncate(message, 500)
			if err := m.PostStatus(message); err != nil {
				log.Println(err)
				m = nil
			} else {
				log.Println("Posted to Mastodon: " + message)
				config.Store.State.LastPostedFoundTime = gc.LastFoundTime
				if err := config.Save(); err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
			}

		}
		time.Sleep(1 * time.Minute)
	}

}
