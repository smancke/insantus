package main

import (
	"log"
)

var store *Store

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("error reading configuration %v\n", err)
	}

	store, err = NewStore(cfg, NewNotificationGateway(cfg))
	if err != nil {
		log.Fatalf("error opening database %v\n", err)
	}

	resultCallback := make(chan []Result, 50)
	startChecking(cfg, resultCallback)
	httpServer := NewHttpServer(cfg, store)
	go httpServer.Start()

	for results := range resultCallback {
		for _, result := range results {
			err := store.InsertResult(result)
			if err != nil {
				log.Printf("error storing check result: %v\n", err)
			}
			if result.Status == StatusUp {
				log.Printf("%v: %v/%v (%vms)\n", result.Status, result.Environment, result.Check, result.Duration)
			} else {
				log.Printf("%v: %v/%v (%vms): %v\n", result.Status, result.Environment, result.Check, result.Duration, result.Message)
			}
		}
	}
}
