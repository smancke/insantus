package main

import (
	"log"
	"net/http"
)

func startHttpServer(cfg *Config, store *Store) {
	log.Printf("starting http server at: %v\n", cfg.Listen)
	err := http.ListenAndServe(cfg.Listen, nil)
	if err != nil {
		log.Fatalf("error starting http service %v\n", err)
	}
}
