package handler

import "net/http"

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	writeText(w, http.StatusOK, "OK")
}
