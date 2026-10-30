package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/utils"
	"github.com/go-chi/chi/v5"
)

func GetCertificationsHandler(store db.CertificationStore, kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var result any
		var err error
		if kind == "milestones" {
			result, err = store.GetMilestoneHistory(r.Context())
		} else {
			id, parseErr := strconv.ParseInt(chi.URLParam(r, "id"), 10, 32)
			if parseErr != nil || id <= 0 {
				utils.WriteError(w, "invalid ID", http.StatusBadRequest)
				return
			}
			if kind == "track" {
				result, err = store.GetTrackCertifications(r.Context(), int32(id))
			} else {
				result, err = store.GetArtistCertifications(r.Context(), int32(id))
			}
		}
		if errors.Is(err, db.ErrNotFound) {
			utils.WriteError(w, "item not found", http.StatusNotFound)
			return
		}
		if err != nil {
			logger.FromContext(r.Context()).Err(err).Msg("Failed to retrieve certifications")
			utils.WriteError(w, "failed to retrieve certifications", http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, http.StatusOK, result)
	}
}
