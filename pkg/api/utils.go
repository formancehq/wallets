package api

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/go-libs/sharedlogging"
)

func notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func internalError(w http.ResponseWriter, r *http.Request, err error) {
	sharedlogging.GetLogger(r.Context()).Error(err)
	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(map[string]string{
		// @todo: return a proper error
		"error": err.Error(),
	}); err != nil {
		panic(err)
	}
}

func created(w http.ResponseWriter, v any) {
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}
