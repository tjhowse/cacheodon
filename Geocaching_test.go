package main

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestUpdate(t *testing.T) {
	var err error
	tempdir := t.TempDir()
	conf := configStore{
		DBFilename: tempdir + "/test.sqlite3",
	}
	var g *Geocaching
	if g, err = NewGeocaching(conf); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer g.Close()
}
