package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/formancehq/go-libs/sharedapi"
	"github.com/formancehq/go-libs/sharedlogging"
)

func notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func noContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func badRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
		ErrorCode:    "INTERNAL_ERROR",
		ErrorMessage: err.Error(),
	}); err != nil {
		panic(err)
	}
}

func internalError(w http.ResponseWriter, r *http.Request, err error) {
	sharedlogging.GetLogger(r.Context()).Error(err)

	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
		ErrorCode:    "INTERNAL_ERROR",
		ErrorMessage: err.Error(),
	}); err != nil {
		panic(err)
	}
}

func created(w http.ResponseWriter, v any) {
	w.WriteHeader(http.StatusCreated)
	ok(w, v)
}

func ok(w io.Writer, v any) {
	if err := json.NewEncoder(w).Encode(sharedapi.BaseResponse[any]{
		Data: &v,
	}); err != nil {
		panic(err)
	}
}
