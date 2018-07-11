package main

import (
	"net/http"
)

var msg = `{
    "service UP": {
        "name": "service UP",
        "status": "UP",
        "message": "This service is up ",
        "timestamp": "2018-07-06T07:58:25.631283402Z"
    },
    "service DEGRADED": {
        "name": "service DEGRADED",
        "status": "DEGRADED",
        "message": "This service is degraded ",
        "timestamp": "2018-07-06T07:58:25.631283402Z"
    },
    "service DOWN": {
        "name": "service DOWN",
        "status": "DOWN",
        "message": "This service is down ",
        "timestamp": "2018-07-06T07:58:25.631283402Z"
    }
}`

func detailedHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)

	w.Write([]byte(msg))
}

func main() {
	http.HandleFunc("/health", detailedHealth)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		panic(err)
	}
}
