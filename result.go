package main

import (
	"time"
)

type Status string

var (
	StatusUp       = Status("UP")
	StatusDown     = Status("DOWN")
	StatusDegraded = Status("DEGRADED")
	StatusError    = Status("ERROR")
)

type Result struct {
	Id          uint   `gorm:"primary_key"`
	Environment string `sql:"type:varchar(50);index"`
	Check       string `sql:"type:varchar(50);index"`
	Name        string
	Status      Status
	Message     string
	Detail      string
	Duration    int
	Timestamp   time.Time
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
		Status:      StatusError,
		Message:     err.Error(),
		Timestamp:   time.Now(),
	}
}
