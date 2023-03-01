package main

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// This stores the detail of a single find event.
type CacheFind struct {
	gorm.Model
	Name      string
	FindTime  time.Time
	CacheCode string
	LogString string
}

type Cache struct {
	gorm.Model
	Code       string
	PlacedTime time.Time
}

// This stores the finder database.
type FinderDB struct {
	db *gorm.DB
}

// Open the DB and migrate if required.
func (f *FinderDB) Init(filename string) error {
	var err error
	f.db, err = gorm.Open(sqlite.Open(filename), &gorm.Config{})
	if err != nil {
		return err
	}

	// Migrate the schema
	f.db.AutoMigrate(&CacheFind{})
	f.db.AutoMigrate(&Cache{})

	return nil
}

// Close the DB.
func (f *FinderDB) Close() error {
	sqlDB, err := f.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// TODO If a new cache shows up in the database publish a message about it.

// This adds a cache to the database
func (f *FinderDB) AddCache(gc Geocache) error {
	if t, err := parseTime(gc.PlacedDate); err == nil {
		f.db.Create(&Cache{Code: gc.Code, PlacedTime: t})
		return nil
	} else {
		return err
	}
}

// This adds a find to the database
func (f *FinderDB) AddFind(name string, findTime time.Time, cacheCode string, logString string) {
	f.db.Create(&CacheFind{Name: name, FindTime: findTime, CacheCode: cacheCode, LogString: logString})
}

// This returns the number of finds since local midnight for a given name.
func (f *FinderDB) FindsSinceMidnight(name string) int {
	return f.FindsSinceTime(name, time.Now().Truncate(24*time.Hour))
}

// This returns the number of finds since local midnight for a given name.
func (f *FinderDB) FindsSinceTime(name string, t time.Time) int {
	var count int64
	f.db.Model(&CacheFind{}).Where("name = ? AND find_time >= ?", name, t).Count(&count)
	return int(count)
}

func NewFinderDB(filename string) (*FinderDB, error) {
	fdb := &FinderDB{}
	if err := fdb.Init(filename); err != nil {
		return fdb, err
	}

	return fdb, nil
}
