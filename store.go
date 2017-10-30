package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

type Store struct {
	db *gorm.DB
}

func NewStore(cfg *Config) (*Store, error) {
	filename := cfg.DBPath
	log.Printf("opening sqlite3 db: %v", filename)
	gormdb, err := gorm.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	err = gormdb.DB().Ping()
	if err != nil {
		log.Printf("error pinging database: %v\n", err)
		return nil, err
	}

	//gormdb.LogMode(true)
	gormdb.DB().SetMaxIdleConns(2)
	gormdb.DB().SetMaxOpenConns(cfg.Worker + 1)
	gormdb.SingularTable(true)

	err = gormdb.AutoMigrate(&Result{}).Error
	if err != nil {
		log.Printf("error in schema migration: %v", err)
		return nil, err
	}

	return &Store{
		db: gormdb,
	}, nil
}

func (store *Store) InsertResult(result Result) error {
	return store.db.Create(&result).Error
}

func (store *Store) GetLatestResults(environment string) (results []*Result, err error) {
	err = store.db.
		Where(`environment = ?`, environment).
		Group(`"check"`).
		Order("timestamp ASC").
		Find(&results).
		Error

	return
}

func (store *Store) CountGoodAndBad(results []*Result) (good, bad int) {
	for _, res := range results {
		if res.Status == StatusUp {
			good++
		} else {
			bad++
		}
	}
	return
}

type StatusChange struct {
	Environment string
	Check       string
	StatusFrom  string
	StatusTo    string
	Time        time.Time
	ResultId    uint
}
