package main

import (
	"os"
	"time"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

type GeocachingAPIer interface {
	Auth(clientID, clientSecret string) error
	Search(st searchTerms) ([]Geocache, error)
	GetLogs(geocache *Geocache) ([]GeocacheLog, error)
}

type Geocaching struct {
	api  GeocachingAPIer
	db   *FinderDB
	conf configStore
}

func NewGeocaching(conf configStore, api GeocachingAPIer) (*Geocaching, error) {
	var err error
	g := &Geocaching{}
	g.conf = conf
	g.api = api
	if err = g.api.Auth(os.Getenv("GEOCACHING_CLIENT_ID"), os.Getenv("GEOCACHING_CLIENT_SECRET")); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	g.db, err = NewFinderDB(conf.DBFilename)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	return g, nil
}

func (g *Geocaching) Close() {
	g.db.Close()
}

// This polls the API for a list of geocaches and updates our database
// with the results. It returns a slice of strings containg the posts
// detailing the new or updated geocaches that should be posted to
// mastodon.
func (g *Geocaching) Update() ([]string, error) {
	var results []string

	caches, err := g.api.Search(g.conf.SearchTerms)
	if err != nil {
		return results, err
	}
	log.Println("Found", len(caches), "geocaches")
	for _, cache := range caches {
		new, updated := g.db.UpdateCache(&cache)
		if new {
			log.Printf("New cache: %+v", cache.Name)
			if post, err := g.WriteNewCachePost(&cache); err == nil {
				results = append(results, post)
			} else {
				log.Error(err)
			}
		}
		if updated {
			log.Printf("Updated cache: %+v", cache.Name)
			if post, err := g.WriteFoundPost(&cache); err == nil {
				results = append(results, post)
			} else {
				log.Error(err)
			}
		}
	}
	return results, nil
}

// This truncates a string to the given maximum length and returns
// the result. If truncation was necessary, it adds an elipsis to
// the end of the string.
func truncate(s string, max int) string {
	if len(s) >= max {
		return s[:max-1] + "…"
	}
	return s
}

func (g *Geocaching) WriteFoundPost(gc *Geocache) (string, error) {
	var err error
	var logs []GeocacheLog
	if logs, err = g.GetLogs(gc); err != nil {
		return "", err
	}
	g.db.AddLog(&logs[0], gc)
	message := ""
	message += "In " + g.conf.SearchTerms.AreaName + ", \"" + logs[0].UserName + "\""
	message += " just found the \"" + gc.Name + "\" geocache! https://www.geocaching.com" + gc.DetailsURL
	if findCount := g.db.FindsSinceMidnight(logs[0].UserName); findCount > 1 {
		message += " That's their " + humanize.Ordinal(findCount) + " find today!"
	}
	message += " They wrote: \"" + logs[0].LogText + "\""

	geocachingHashtagString := " #geocaching"
	message = truncate(message, 500-len(geocachingHashtagString))
	message += geocachingHashtagString

	return message, nil
}

func (g *Geocaching) WriteNewCachePost(gc *Geocache) (string, error) {
	return "", nil
}

func (g *Geocaching) GetLogs(geocache *Geocache) ([]GeocacheLog, error) {
	return g.api.GetLogs(geocache)
}

// This returns all geocaches with a LastFoundDate later than the given date
func (g *Geocaching) SearchSince(st searchTerms, since time.Time) ([]Geocache, error) {
	var err error
	var results []Geocache

	if results, err = g.api.Search(st); err != nil {
		return nil, err
	}

	return FilterFoundSince(results, since), nil
}

// This filters a []Geocache to only include those that have a LastFoundDate later than the given date
func FilterFoundSince(geocaches []Geocache, since time.Time) []Geocache {
	// The results are sorted by LastFoundDate, so we can just iterate backwards until we find
	// the first result that is before the given date.
	for i := len(geocaches) - 1; i >= 0; i-- {
		if geocaches[i].LastFoundTime.Before(since) || geocaches[i].LastFoundTime.Equal(since) {
			return geocaches[i+1:]
		}
	}
	return []Geocache{}
}

// Returns a slice of geocaches that have been updated since we last checked.
func SearchUpdated(st searchTerms) ([]Geocache, error) {
	return nil, nil
}
