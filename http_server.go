package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"regexp"
	"time"
)

type HttpServer struct {
	cfg   *Config
	store *Store
}

func NewHttpServer(cfg *Config, store *Store) *HttpServer {
	return &HttpServer{
		cfg:   cfg,
		store: store,
	}
}

func (server *HttpServer) Start() {
	log.Printf("starting http server at: %v\n", server.cfg.Listen)

	router := mux.NewRouter()
	router.HandleFunc("/api/environments", server.GetEnvironments)
	router.HandleFunc("/api/environments/{env}/checks", server.GetEnvironmentResults)
	router.Handle(`/{path:[a-zA-Z0-9=\-\/.]*}`, http.FileServer(http.Dir(server.cfg.Static)))

	err := http.ListenAndServe(server.cfg.Listen, router)
	if err != nil {
		log.Fatalf("error starting http service %v\n", err)
	}
}

func (server *HttpServer) GetEnvironments(w http.ResponseWriter, r *http.Request) {
	environments := []map[string]interface{}{}
	for _, env := range server.cfg.Environments {
		envInfo := map[string]interface{}{
			"id":      env.Id,
			"name":    env.Name,
			"default": env.Default,
		}

		results, err := server.store.GetLatestResults(env.Id)
		if err != nil {
			errorResponse(w, err)
			return
		}
		good, bad := store.CountGoodAndBad(results)
		envInfo["good"] = good
		envInfo["bad"] = bad

		if bad > 0 {
			envInfo["status"] = StatusDown
		} else {
			envInfo["status"] = StatusUp
		}

		environments = append(environments, envInfo)
	}
	jsonReponse(w, environments)
}

func (server *HttpServer) GetEnvironmentResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	env := vars["env"]
	results, err := store.GetLatestResults(env)
	if err != nil {
		errorResponse(w, err)
		return
	}

	response := []map[string]interface{}{}
	for _, r := range results {
		info := map[string]interface{}{
			"check":      r.Check,
			"name":       r.Name,
			"status":     r.Status,
			"message":    r.Message,
			"detail":     r.Detail,
			"duration":   r.Duration,
			"sinceCheck": since(r.Timestamp),
		}
		response = append(response, info)
	}
	jsonReponse(w, response)
}

func errorResponse(w http.ResponseWriter, err error) {
	log.Printf("internal error occured %v\n", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)
	w.Write([]byte(`{"error": "Internal error occured"}`))
}

func jsonReponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	err := enc.Encode(data)
	if err != nil {
		log.Printf("error writing json %v\n", err)
	}
}

var sinceRegex = regexp.MustCompile("\\.[0-9]+s")

func since(t time.Time) string {
	s := time.Since(t).String()
	return sinceRegex.ReplaceAllString(s, "s")
}
