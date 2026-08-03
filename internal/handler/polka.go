package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

type PolkaHandler struct {
	apiConfig *ApiConfig
	store     polkaStore
}

type polkaStore interface {
	UpdateChirpyRedFlag(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error)
}

func NewPolkaHandler(apiConfig *ApiConfig) *PolkaHandler {
	return &PolkaHandler{apiConfig: apiConfig, store: apiConfig.Database}
}

func (h *PolkaHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/api/polka/webhooks", h.apiConfig.RequirePolkaAuth(http.HandlerFunc(h.PolkaWebhook)))
}

func (h *PolkaHandler) PolkaWebhook(w http.ResponseWriter, r *http.Request) {
	type PolkaWebhookRequest struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	var req PolkaWebhookRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.apiConfig.Logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Event != "user.upgraded" {
		writeNoContent(w)
		return
	}

	userID, err := uuid.Parse(req.Data.UserID)
	if err != nil {
		h.apiConfig.Logger.Error("Invalid user ID", err)
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	_, err = h.store.UpdateChirpyRedFlag(r.Context(), database.UpdateChirpyRedFlagParams{
		ID:          userID,
		IsChirpyRed: true,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "User not found")
			return
		}

		h.apiConfig.Logger.Error("Failed to update user", err)
		writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	writeNoContent(w)
}
