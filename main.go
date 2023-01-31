package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/health", HealthHandler).Methods("GET")
	r.HandleFunc("/api/v1/health", HealthHandler).Methods("POST")
	http.ListenAndServe(":8080", r)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}
