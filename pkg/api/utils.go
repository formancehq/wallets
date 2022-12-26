package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/formancehq/go-libs/sharedapi"
	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/wallets/pkg/wallet"
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

func cursor[T any](w io.Writer, v sharedapi.Cursor[T]) {
	if err := json.NewEncoder(w).Encode(sharedapi.BaseResponse[T]{
		Cursor: &v,
	}); err != nil {
		panic(err)
	}
}

func cursorFromListResponse[T any, V any](w io.Writer, query wallet.ListQuery[V], response *wallet.ListResponse[T]) {
	cursor(w, sharedapi.Cursor[T]{
		PageSize: query.Limit,
		HasMore:  response.HasMore,
		Previous: response.Previous,
		Next:     response.Next,
		Data:     response.Data,
	})
}

func parsePaginationToken(r *http.Request) string {
	return r.URL.Query().Get("cursor")
}

const defaultLimit = 15

func parseLimit(r *http.Request) int {
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		return defaultLimit
	}

	v, err := strconv.ParseInt(limit, 10, 32)
	if err != nil {
		panic(err)
	}
	return int(v)
}
