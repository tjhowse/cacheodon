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
		db.AddLog("testname", timeNow, "GC123", "testlog")
		// Check one find in the last 24 hours.
		if want, got := 1, db.FindsSinceTime("testname", timeMidnight); want != got {
			t.Fatalf("FindsSinceMidnight returned wrong value: want %d, got %d", want, got)
		}
		// Add an irrelevant find.
		db.AddLog("testname2", timeNow, "GC321", "testlog")
		// Add another find ten minutes ago
		db.AddLog("testname", timeNow.Add(-10*time.Minute), "GC456", "testlog2")
		// Check we now have two finds since midnight
		if want, got := 2, db.FindsSinceTime("testname", timeMidnight); want != got {
			t.Fatalf("FindsSinceMidnight returned wrong value: want %d, got %d", want, got)
		}
		// Add a find 24 hours ago and ensure it doesn't count towards today's finds
		db.AddLog("testname", timeNow.Add(-24*time.Hour), "GC789", "testlog3")
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

	gc := Geocache{
		Code:       "GC123",
		PlacedDate: "2023-03-02T16:44:59",
	}

	// Create an empty DB
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Close()
		db.AddLog("testname", timeNow, "GC123", "testlog")
		db.AddCache(gc)
	}
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Close()
		if want, got := 1, db.FindsSinceTime("testname", timeMidnight); want != got {
			t.Fatalf("FindsSinceMidnight returned wrong value: want %d, got %d", want, got)
		}
		if new, err := db.AddCache(gc); err != nil {
			t.Fatal(err)
		} else {
			if new {
				t.Fatal("Cache should not be new")
			}
		}
	}
}

func TestAddCache(t *testing.T) {
	tempdir := t.TempDir()
	// Create an empty DB
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Close()
		// Add a cache
		gc := Geocache{
			Code:       "GC123",
			PlacedDate: "2023-03-02T16:44:59",
		}
		if new, err := db.AddCache(gc); err != nil {
			t.Fatal(err)
		} else {
			if !new {
				t.Fatal("Cache should be new")
			}
		}
		// Check the cache is there

		if new, err := db.AddCache(gc); err != nil {
			t.Fatal(err)
		} else {
			if new {
				t.Fatal("Cache should not be new")
			}
		}
		gc = Geocache{
			Code: "GC321",
			// Deliberately bad timestamp
			PlacedDate: "202asdasdfasdf02T16:44:59",
		}
		if new, err := db.AddCache(gc); err == nil || new {
			t.Fatal("Should have failed to add cache due to bad timestamp")
		}

	}
}

func TestLastPostedFoundTime(t *testing.T) {
	tempdir := t.TempDir()
	// Set the "current time" to midday so we don't run into issues with the midnight rollover.
	timeNow := time.Date(2021, 1, 1, 12, 0, 0, 0, time.UTC)
	timeLater := time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create an empty DB
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Close()
		db.SetLastPostedFoundTime(timeNow)
		if want, got := timeNow, db.GetLastPostedFoundTime(timeLater); want != got {
			t.Fatalf("LastPostedFoundTime returned wrong value: want %s, got %s", want, got)
		}
	}
	// Check the value is persisted
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Close()
		if want, got := timeNow, db.GetLastPostedFoundTime(timeLater); want != got {
			t.Fatalf("LastPostedFoundTime didn't remember the right value: want %s, got %s", want, got)
		}
	}
}

func TestLastPostedFoundTimeDefault(t *testing.T) {
	tempdir := t.TempDir()
	// Set the "current time" to midday so we don't run into issues with the midnight rollover.
	timeNow := time.Date(2021, 1, 1, 12, 0, 0, 0, time.UTC)
	timeLater := time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create an empty DB
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Close()
		if want, got := timeLater, db.GetLastPostedFoundTime(timeLater); want != got {
			t.Fatalf("LastPostedFoundTime returned wrong value: want %s, got %s", want, got)
		}
	}
	// Check the value is persisted
	if db, err := NewFinderDB(tempdir + "/test.sqlite3"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Close()
		if want, got := timeLater, db.GetLastPostedFoundTime(timeNow); want != got {
			t.Fatalf("LastPostedFoundTime didn't remember the right value: want %s, got %s", want, got)
		}
	}
}
