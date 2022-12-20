package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (m *MainHandler) GetHoldHandler(w http.ResponseWriter, r *http.Request) {
	hold, err := m.repository.GetHold(r.Context(), chi.URLParam(r, "hold_id"))
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			// @todo: return a proper error
			"error": err.Error(),
		})
		return
	}

	render.JSON(w, r, hold)
}
