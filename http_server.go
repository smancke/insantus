package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
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
	router.HandleFunc("/api/environments/{env}", server.GetEnvironment)
	router.HandleFunc("/api/results/{id}", server.GetResult)
	router.Handle(`/{path:[a-zA-Z0-9=\-\/.]*}`, http.FileServer(http.Dir(server.cfg.Static)))

	err := http.ListenAndServe(server.cfg.Listen, router)
	if err != nil {
		log.Fatalf("error starting http service %v\n", err)
	}
}

func (server *HttpServer) GetResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		badRequestResponse(w)
		return
	}

	res, found, err := store.Result(id)
	if err != nil {
		errorResponse(w, err)
		return
	}

	if !found {
		w.WriteHeader(404)
		return
	}
	jsonReponse(w, res)
}
func (server *HttpServer) GetEnvironments(w http.ResponseWriter, r *http.Request) {
	overallStatus := StatusUp
	response := map[string]interface{}{}
	for _, env := range server.cfg.Environments {
		envInfo := map[string]interface{}{
			"id":      env.Id,
			"name":    env.Name,
			"default": env.Default,
		}

		status, err := server.store.Status(env.Id)
		if err != nil {
			errorResponse(w, err)
			return
		}
		good, bad := store.CountGoodAndBad(status)
		envInfo["good"] = good
		envInfo["bad"] = bad

		if bad > 0 {
			envInfo["status"] = StatusDown
			overallStatus = StatusDown
		} else {
			envInfo["status"] = StatusUp
		}

		response[env.Id] = envInfo
	}
	response["status"] = overallStatus
	jsonReponse(w, response)
}

func (server *HttpServer) GetEnvironment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	env := vars["env"]
	status, err := store.Status(env)
	if err != nil {
		errorResponse(w, err)
		return
	}

	overallStatus := StatusUp
	checks := []map[string]interface{}{}
	for _, s := range status {
		if s.Status != StatusUp {
			overallStatus = StatusDown
		}

		info := map[string]interface{}{
			"check":        s.Check,
			"name":         s.Name,
			"status":       s.Status,
			"message":      s.Message,
			"duration":     s.Duration,
			"lastResultId": s.LastResultId,
			"time":         s.Updated,
			"sinceCheck":   sinceMs(s.Updated),
		}

		if s.Detail != "" {
			jsonDetails := map[string]interface{}{}
			err := json.Unmarshal([]byte(s.Detail), &jsonDetails)
			if err == nil {
				info["detail"] = jsonDetails
			}
		}

		checks = append(checks, info)
	}

	downtimes, err := server.store.Downtimes(env)
	if err != nil {
		errorResponse(w, err)
		return
	}

	response := map[string]interface{}{
		"status":    overallStatus,
		"checks":    checks,
		"downtimes": downtimes,
	}

	jsonReponse(w, response)
}

func errorResponse(w http.ResponseWriter, err error) {
	log.Printf("internal error occured %v\n", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)
	w.Write([]byte(`{"error": "Internal error occured"}`))
}

func badRequestResponse(w http.ResponseWriter) {
	w.WriteHeader(400)
	w.Write([]byte(`{"error": "Bad Request"}`))
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

func sinceMs(t time.Time) int64 {
	return int64(time.Since(t) / time.Millisecond)
}
