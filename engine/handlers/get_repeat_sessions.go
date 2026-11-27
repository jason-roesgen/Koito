package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/utils"
)

func GetRepeatSessionsHandler(store db.RepeatSessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query()
		opts := db.RepeatSessionOpts{
			Query: strings.TrimSpace(p.Get("q")), Sort: p.Get("sort"), Order: p.Get("order"), Page: 1,
		}
		if page := p.Get("page"); page != "" {
			n, err := strconv.Atoi(page)
			if err != nil || n < 1 || n > 1_000_000 {
				utils.WriteError(w, "invalid page", http.StatusBadRequest)
				return
			}
			opts.Page = n
		}
		if utf8.RuneCountInString(opts.Query) > 200 ||
			(opts.Sort != "" && opts.Sort != "plays" && opts.Sort != "date" && opts.Sort != "time") ||
			(opts.Order != "" && opts.Order != "asc" && opts.Order != "desc") {
			utils.WriteError(w, "invalid repeat session filters", http.StatusBadRequest)
			return
		}
		for _, bound := range []struct {
			name string
			dest **int64
		}{{"from", &opts.From}, {"to", &opts.To}} {
			if raw := p.Get(bound.name); raw != "" {
				value, err := strconv.ParseInt(raw, 10, 64)
				if err != nil {
					utils.WriteError(w, "invalid date range", http.StatusBadRequest)
					return
				}
				*bound.dest = &value
			}
		}
		if opts.From != nil && opts.To != nil && *opts.From >= *opts.To {
			utils.WriteError(w, "invalid date range", http.StatusBadRequest)
			return
		}

		result, err := store.GetRepeatSessions(r.Context(), opts)
		if err != nil {
			logger.FromContext(r.Context()).Err(err).Msg("Failed to retrieve repeat sessions")
			utils.WriteError(w, "failed to retrieve repeat sessions", http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, http.StatusOK, result)
	}
}
