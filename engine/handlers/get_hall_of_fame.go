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

func GetHallOfFameHandler(store db.HallOfFameStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query()
		opts := db.HallOfFameOpts{View: p.Get("view"), Query: strings.TrimSpace(p.Get("q")), Tier: p.Get("tier"), Sort: p.Get("sort"), Order: p.Get("order"), Page: 1}
		if opts.View == "" {
			opts.View = "artists"
		}
		if opts.Sort == "" {
			opts.Sort = "count"
		}
		if opts.Order == "" {
			opts.Order = "desc"
		}
		valid := func(value string, allowed ...string) bool {
			for _, a := range allowed {
				if a == value {
					return true
				}
			}
			return false
		}
		if page := p.Get("page"); page != "" {
			n, err := strconv.Atoi(page)
			if err != nil || n < 1 || n > 1000000 {
				utils.WriteError(w, "invalid page", 400)
				return
			}
			opts.Page = n
		}
		if !valid(opts.View, "songs", "artists") || !valid(opts.Tier, "", "gold", "platinum", "diamond") || !valid(opts.Sort, "count", "name", "date") || !valid(opts.Order, "asc", "desc") || utf8.RuneCountInString(opts.Query) > 200 {
			utils.WriteError(w, "invalid Hall of Fame filters", 400)
			return
		}
		result, err := store.GetHallOfFame(r.Context(), opts)
		if err != nil {
			logger.FromContext(r.Context()).Err(err).Msg("Failed to retrieve Hall of Fame")
			utils.WriteError(w, "failed to retrieve Hall of Fame", 500)
			return
		}
		utils.WriteJSON(w, http.StatusOK, result)
	}
}
