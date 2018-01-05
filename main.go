package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

var store *Store

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("error reading configuration %v\n", err)
	}

	profiler(cfg)

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

func profiler(cfg *Config) {
	if !cfg.Pprof {
		return
	}
	pprofMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	go func() {
		log.Fatal(http.ListenAndServe(cfg.PprofListen, pprofMux))
	}()
}
