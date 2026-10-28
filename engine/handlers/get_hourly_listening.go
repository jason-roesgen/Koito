package handlers

import (
	"net/http"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/utils"
)

func GetHourlyListeningHandler(store db.HourlyListeningStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		period := r.URL.Query().Get("period")
		switch db.Period(period) {
		case db.PeriodDay, db.PeriodWeek, db.PeriodMonth, db.PeriodYear, db.PeriodAllTime:
		default:
			utils.WriteError(w, "period must be day, week, month, year, or all_time", http.StatusBadRequest)
			return
		}
		timeframe := TimeframeFromRequest(r)
		buckets, err := store.GetHourlyListening(r.Context(), timeframe)
		if err != nil {
			logger.FromContext(r.Context()).Err(err).Msg("Failed to retrieve hourly listening")
			utils.WriteError(w, "failed to retrieve hourly listening", http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, http.StatusOK, struct {
			Timezone string                     `json:"timezone"`
			Hours    []db.HourlyListeningBucket `json:"hours"`
		}{Timezone: timeframe.Timezone.String(), Hours: buckets})
	}
}
