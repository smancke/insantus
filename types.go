package main

import (
	"time"
)

var (
	StatusUp       = "UP"
	StatusDown     = "DOWN"
	StatusDegraded = "DEGRADED"
)

type Result struct {
	Id          uint   `gorm:"primary_key"`
	Environment string `sql:"type:varchar(50);index"`
	Check       string `sql:"type:varchar(50);index"`
	Name        string
	Status      string
	Message     string
	Detail      string
	Duration    int
	Timestamp   time.Time `sql:"index"`
}

func NewResult(environment, check, name string) Result {
	return Result{
		Environment: environment,
		Check:       check,
		Name:        name,
		Status:      StatusUp,
		Timestamp:   time.Now(),
	}
}

func NewResultFromError(environment, check, name string, err error) Result {
	return Result{
		Environment: environment,
		Check:       check,
		Name:        name,
		Status:      StatusDown,
		Message:     err.Error(),
		Timestamp:   time.Now(),
	}
}

type CheckStatus struct {
	Environment  string    `json:"environment" gorm:"primary_key" sql:"type:varchar(50)"`
	Check        string    `json:"check" gorm:"primary_key" sql:"type:varchar(50)"`
	Name         string    `json:"name"`
	Status       string    `json:"status" sql:"type:varchar(50);index"`
	Message      string    `json:"message"`
	Detail       string    `json:"detail"`
	Duration     int       `json:"duration"`
	LastResultId uint      `json:"lastResultId"`
	Updated      time.Time `json:"updated" sql:"index"`
}

type Downtime struct {
	Id                uint      `json:"id" gorm:"primary_key"`
	Environment       string    `json:"environment" sql:"type:varchar(50);index"`
	Check             string    `json:"check" sql:"type:varchar(50);index"`
	Name              string    `json:"name"`
	Message           string    `json:"message"`
	Start             time.Time `json:"start"`
	End               time.Time `json:"end"`
	FailCount         int       `json:"failCount"`
	LastResultId      uint      `json:"lastResultId"`
	Recovered         bool      `json:"recovered" sql:"index"`
	Comment           string    `json:"comment"`
	DownNotifySent    bool      `json:"downNotifySent"`
	DownNotifyTime    time.Time `json:"downNotifyTime"`
	RecoverNotifySent bool      `json:"recoverNotifySent"`
	RecoverNotifyTime time.Time `json:"recoverNotifyTime"`
}

type Notifyer interface {
	NotifyDown(envId string, downtimes []*Downtime) error
	NotifyRecovered(envId string, downtimes []*Downtime) error
}
