package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type checkJob struct {
	checker Checker
	done    chan bool
}

func startChecking(cfg *Config, resultCallback chan []Result) {
	checkQueue := make(chan checkJob, 50)
	for _, e := range cfg.Environments {
		for _, c := range e.Checks {
			if len(c.Envs) > 0 && !contains(c.Envs, e.Id) {
				continue
			}
			var checker Checker
			var err error
			switch c.Type {
			case "http":
				checker, err = NewHttpCheck(e.Id, c.Id, c.Name, c.Params)
				if err != nil {
					log.Fatalf("error creating check %v/%v: %v\n", e.Id, c.Id, err)
				}
			default:
				log.Fatalf("error creating check for %v/%v: no such type %v\n", e.Id, c.Id, c.Type)
			}
			d := cfg.Duration
			if c.Every != 0 {
				d = c.Every
			}
			go shedule(checker, d, checkQueue)
		}
	}

	go monitorQueue(cfg, checkQueue)

	for i := 0; i < cfg.Worker; i++ {
		go worker(checkQueue, resultCallback)
	}
}

func monitorQueue(cfg *Config, checkQueue chan checkJob) {
	ticker := time.Tick(time.Second * 20)
	for {
		if len(checkQueue) > cfg.Worker {
			log.Printf("WARNING: queue size: %v", len(checkQueue))
		}
		<-ticker
	}
}

func worker(checkQueue chan checkJob, resultCallback chan []Result) {
	for {
		job := <-checkQueue
		results := job.checker.Check()
		job.done <- true
		resultCallback <- results
	}
}

func shedule(checker Checker, d time.Duration, checkQueue chan checkJob) {
	ticker := time.Tick(d)
	for {
		done := make(chan bool)
		checkQueue <- checkJob{checker, done}
		<-done
		<-ticker
	}
}

func dumpJson(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	err := enc.Encode(data)
	if err != nil {
		log.Fatalf("error dumping json %v\n", err)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
