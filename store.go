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

	err = gormdb.AutoMigrate(&Result{}, &CheckInfo{}, &StatusEvent{}).Error
	if err != nil {
		return nil, errors.Wrap(err, "schema migration")
	}

	s := &Store{
		db: gormdb,
	}

	err = s.updateCheckInfos(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "updating CheckSummaries")
	}

	return s, nil
}

func (store *Store) updateCheckInfos(cfg *Config) error {
	allKeysInConfig := map[string]string{}
	for _, e := range cfg.Environments {
		for _, c := range e.Checks {
			info := &CheckInfo{}
			if store.db.Where(`environment = ? AND "check" = ?`, e.Id, c.Id).First(info).RecordNotFound() {
				// nothing found: insert record
				info := &CheckInfo{
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

	allInfosInDb := []CheckInfo{}
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

	return store.updateCheckInfo(result)
}

func (store *Store) updateCheckInfo(result Result) error {
	info := CheckInfo{
		Environment: result.Environment,
		Check:       result.Check,
	}
	err := store.db.Where(&info).First(&info).Error
	if err != nil {
		return errors.Wrap(err, "query check info")
	}

	statusFrom := info.Status
	statusTo := result.Status

	info.Status = result.Status
	info.Message = result.Message
	info.Duration = result.Duration
	info.Updated = result.Timestamp

	err = store.db.Where(`environment = ? AND "check" = ?`, info.Environment, info.Check).Save(&info).Error
	if err != nil {
		return errors.Wrap(err, "update check info")
	}

	if statusFrom != statusTo {
		event := StatusEvent{
			Environment: info.Environment,
			Check:       info.Check,
			Name:        info.Name,
			StatusFrom:  statusFrom,
			StatusTo:    statusTo,
			Updated:     info.Updated,
			ResultId:    result.Id,
		}
		err := store.db.Create(&event).Error
		if err != nil {
			return errors.Wrap(err, "create status event")
		}
	}

	return nil
	/**
	DurationAvg1h    int
	DurationAvg24h   int
	DurationAvg7d    int
	UptimePercent5m    float32
	UptimePercent1h  float32
	UptimePercent24h float32
	UptimePercent7d  float32
	*/
}

func (store *Store) GetStatusEvents(environment string) (results []*StatusEvent, err error) {
	err = store.db.
		Where(`environment = ?`, environment).
		Order("updated DESC").
		Limit(30).
		Find(&results).
		Error

	return
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

type CheckInfo struct {
	Environment      string    `json:"environment" gorm:"primary_key" sql:"type:varchar(50)"`
	Check            string    `json:"check" "gorm:"primary_key" sql:"type:varchar(50)"`
	Name             string    `json:"name"`
	Status           string    `json:"status" sql:"type:varchar(50);index"`
	Message          string    `json:"message"`
	Duration         int       `json:"duration"`
	DurationAvg1h    int       `json:"durationAvg1h"`
	DurationAvg24h   int       `json:"durationAvg24h"`
	DurationAvg7d    int       `json:"durationAvg7dh"`
	UptimePercent5m  float32   `json:"uptimePercent5m"`
	UptimePercent1h  float32   `json:"uptimePercent1h"`
	UptimePercent24h float32   `json:"uptimePercent24h"`
	UptimePercent7d  float32   `json:"uptimePercent7d"`
	Updated          time.Time `json:"updated" sql:"index"`
}

type StatusEvent struct {
	Environment string    `json:"environment" "sql:"type:varchar(50);index"`
	Check       string    `json:"check" sql:"type:varchar(50);index"`
	Name        string    `json:"name"`
	StatusFrom  string    `json:"statusFrom"`
	StatusTo    string    `json:"statusTo"`
	Updated     time.Time `json:"updated" sql:"index"`
	ResultId    uint      `json:"resultId"`
}
