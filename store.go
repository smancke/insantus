package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type Store struct {
	db *gorm.DB
}

func NewStore(filename string) (*Store, error) {
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
	gormdb.DB().SetMaxOpenConns(5)
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
