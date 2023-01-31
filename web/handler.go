package web

// handler for api server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// HealthHandler is a handler for health check
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

// create member handler is a handler for create member
func CreateMemberHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fmt.Fprintf(w, "CreateMemberHandler: %v", vars["id"])
}
