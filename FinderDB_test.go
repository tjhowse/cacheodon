package main

import (
	"testing"
	"time"
)

func TestInit(t *testing.T) {

	tempdir := t.TempDir()
	// Create an empty DB
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		db.Close()
	}
}

func TestAddFind(t *testing.T) {
	tempdir := t.TempDir()
	// Set the "current time" to midday so we don't run into issues with the midnight rollover.
	timeNow := time.Date(2021, 1, 1, 12, 0, 0, 0, time.UTC)
	timeMidnight := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create an empty DB
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Close()
		// Add a find from right now
		db.AddFind("testname", timeNow, "GC123", "testlog")
		// Check one find in the last 24 hours.
		if want, got := 1, db.FindsSinceTime("testname", timeMidnight); want != got {
			t.Fatalf("FindsSinceMidnight returned wrong value: want %d, got %d", want, got)
		}
		// Add an irrelevant find.
		db.AddFind("testname2", timeNow, "GC321", "testlog")
		// Add another find ten minutes ago
		db.AddFind("testname", timeNow.Add(-10*time.Minute), "GC456", "testlog2")
		// Check we now have two finds since midnight
		if want, got := 2, db.FindsSinceTime("testname", timeMidnight); want != got {
			t.Fatalf("FindsSinceMidnight returned wrong value: want %d, got %d", want, got)
		}
		// Add a find 24 hours ago and ensure it doesn't count towards today's finds
		db.AddFind("testname", timeNow.Add(-24*time.Hour), "GC789", "testlog3")
		if want, got := 2, db.FindsSinceTime("testname", timeMidnight); want != got {
			t.Fatalf("FindsSinceMidnight returned wrong value: want %d, got %d", want, got)
		}
	}
}

func TestPersistence(t *testing.T) {
	tempdir := t.TempDir()
	// Set the "current time" to midday so we don't run into issues with the midnight rollover.
	timeNow := time.Date(2021, 1, 1, 12, 0, 0, 0, time.UTC)
	timeMidnight := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create an empty DB
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		db.AddFind("testname", timeNow, "GC123", "testlog")
		defer db.Close()
	}
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		if want, got := 1, db.FindsSinceTime("testname", timeMidnight); want != got {
			t.Fatalf("FindsSinceMidnight returned wrong value: want %d, got %d", want, got)
		}
		defer db.Close()
	}
}
