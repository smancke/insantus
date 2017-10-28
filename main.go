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

	store, err = NewStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("error opening database %v\n", err)
	}

	startChecking(cfg, resultCallback)
	startHttpServer(cfg, store)
}

func resultCallback(results []Result) {
	for _, result := range results {
		err := store.InsertResult(result)
		if err != nil {
			log.Fatalf("error storing check result: %v\n", err)
		}
		log.Printf("stored result for %v/%v\n", result.Environment, result.Check)
	}
}
