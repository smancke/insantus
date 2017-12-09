package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
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
		return nil, errors.Wrap(err, "pinging database")
	}

	gormdb.LogMode(false)
	gormdb.DB().SetMaxIdleConns(2)
	gormdb.DB().SetMaxOpenConns(cfg.Worker + 1)
	gormdb.SingularTable(true)

	err = gormdb.AutoMigrate(&Result{}, &CheckStatus{}, &Downtime{}).Error
	if err != nil {
		return nil, errors.Wrap(err, "schema migration")
	}

	s := &Store{
		db: gormdb,
	}

	err = s.updateChecks(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "updating CheckSummaries")
	}

	return s, nil
}

func (store *Store) Close() error {
	return store.db.Close()
}

func (store *Store) updateChecks(cfg *Config) error {
	allKeysInConfig := map[string]string{}
	for _, e := range cfg.Environments {
		for _, c := range e.Checks {
			info := &CheckStatus{}
			if store.db.Where(`environment = ? AND "check" = ?`, e.Id, c.Id).First(info).RecordNotFound() {
				// nothing found: insert record
				info := &CheckStatus{
					Environment: e.Id,
					Check:       c.Id,
					Name:        c.Name,
				}
				err := store.db.Create(info).Error
				if err != nil {
					return errors.Wrap(err, "create check info")
				}
			}
			key := e.Id + "/" + c.Id
			allKeysInConfig[key] = key
		}
	}

	allInfosInDb := []CheckStatus{}
	err := store.db.Find(&allInfosInDb).Error
	if err != nil {
		return err
	}
	for _, infoFromDb := range allInfosInDb {
		key := infoFromDb.Environment + "/" + infoFromDb.Check
		if _, existInConfig := allKeysInConfig[key]; !existInConfig {
			err := store.db.Delete(infoFromDb).Error
			if err != nil {
				return errors.Wrap(err, "delete check info")
			}
		}
	}
	return nil
}

func (store *Store) InsertResult(result Result) error {
	err := store.db.Create(&result).Error
	if err != nil {
		return err
	}

	err = store.updateCheckStatus(result)
	if err != nil {
		return err
	}

	err = store.updateDowntimes(result)
	if err != nil {
		return err
	}

	return nil
}

func (store *Store) updateCheckStatus(result Result) error {
	checkStatus := CheckStatus{
		Environment: result.Environment,
		Check:       result.Check,
	}
	err := store.db.Where(&checkStatus).First(&checkStatus).Error
	if err != nil {
		return errors.Wrap(err, "query checkStatus")
	}

	checkStatus.Status = result.Status
	checkStatus.Message = result.Message
	checkStatus.Duration = result.Duration
	checkStatus.Updated = result.Timestamp
	checkStatus.LastResultId = result.Id

	err = store.db.Where(`environment = ? AND "check" = ?`, checkStatus.Environment, checkStatus.Check).
		Save(&checkStatus).Error
	if err != nil {
		return errors.Wrap(err, "update checkStatus")
	}

	return nil
}

func (store *Store) updateDowntimes(result Result) error {
	// load the unrecovered downtime, if any

	d := &Downtime{}
	err := store.db.
		Where(`environment = ? AND "check" = ? AND recovered = 0`, result.Environment, result.Check).
		First(d).Error
	openDowntimeLoaded := err == nil
	if err != nil && err != gorm.ErrRecordNotFound {
		return errors.Wrap(err, "query downtimes")
	}

	if result.Status == StatusUp && !openDowntimeLoaded {
		// nothing to do
		return nil
	}

	if result.Status == StatusUp {
		d.Recovered = true
		d.End = time.Now()
	} else {

		if !openDowntimeLoaded {
			d.Environment = result.Environment
			d.Check = result.Check
			d.Name = result.Name
			d.Start = time.Now()
		}
		d.FailCount++
		d.LastResultId = result.Id
	}

	// TODO: notify

	return store.db.Save(d).Error
}

func (store *Store) Downtimes(environment string) (results []*Downtime, err error) {
	err = store.db.
		Where(`environment = ?`, environment).
		Order("recovered ASC, start DESC").
		Limit(30).
		Find(&results).
		Error

	return
}

func (store *Store) Status(environment string) (statusList []*CheckStatus, err error) {
	err = store.db.
		Where(`environment = ?`, environment).
		Order("name").
		Find(&statusList).
		Error
	return
}

func (store *Store) Result(id int) (*Result, bool, error) {
	res := &Result{}
	err := store.db.First(res, id).
		Error

	found := err == nil
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return res, found, err
}

func (store *Store) CountGoodAndBad(s []*CheckStatus) (good, bad int) {
	for _, res := range s {
		if res.Status == StatusUp {
			good++
		} else {
			bad++
		}
	}
	return
}

/**
func (store *Store) GetLatestResults(environment string) (results []*Result, err error) {
	err = store.db.
		Where(`environment = ?`, environment).
		Group(`"check"`).
		Order("timestamp ASC").
		Find(&results).
		Error

	return
}

**/
