package db

import (
	"cmp"
	"context"
	"sort"
	"strings"
	"time"

	"github.com/gabehf/koito/internal/models"
)

type HallCredit struct {
	ID    int32            `json:"id"`
	Name  string           `json:"name"`
	Image models.ImageList `json:"image"`
}

type HallSong struct {
	TrackID int32            `json:"track_id"`
	Title   string           `json:"title"`
	Artists []HallCredit     `json:"artists"`
	Image   models.ImageList `json:"image"`
	Plays   int64            `json:"plays"`
	Current Certification    `json:"current"`
	Seconds int64            `json:"seconds,omitempty"`
}

type HallArtist struct {
	HallCredit
	Counts         map[string]int `json:"counts"`
	CertifiedSongs int            `json:"certified_songs"`
	ReachedAt      *time.Time     `json:"reached_at"`
}

type HallOfFameOpts struct {
	View, Query, Tier, Sort, Order, Metric string
	Page                                   int
}

type HallOfFame struct {
	Counts           map[string]int `json:"counts"`
	CertifiedSongs   int            `json:"certified_songs"`
	CertifiedArtists int            `json:"certified_artists"`
	Songs            []HallSong     `json:"songs"`
	Artists          []HallArtist   `json:"artists"`
	TotalRecordCount int            `json:"total_record_count"`
	CurrentPage      int            `json:"current_page"`
	ItemsPerPage     int            `json:"items_per_page"`
	HasNextPage      bool           `json:"has_next_page"`
}

type HallOfFameStore interface {
	GetHallOfFame(context.Context, HallOfFameOpts) (*HallOfFame, error)
}

// BuildHallOfFame counts current certificates once per song and once per credited
// artist. Filtering never changes the global counters. Input is one DB snapshot.
func BuildHallOfFame(songs []HallSong, opts HallOfFameOpts) *HallOfFame {
	counts := func() map[string]int { return map[string]int{"gold": 0, "platinum": 0, "diamond": 0} }
	result := &HallOfFame{Counts: counts(), Songs: []HallSong{}, Artists: []HallArtist{}, CurrentPage: opts.Page, ItemsPerPage: 20}
	if result.CurrentPage < 1 {
		result.CurrentPage = 1
	}
	artists := map[int32]*HallArtist{}
	query := strings.ToLower(strings.TrimSpace(opts.Query))
	for _, song := range songs {
		result.Counts[song.Current.Tier]++
		matches := opts.View == "songs" && strings.Contains(strings.ToLower(song.Title), query)
		seen := map[int32]bool{}
		for _, credit := range song.Artists {
			if opts.View == "songs" && !matches {
				matches = strings.Contains(strings.ToLower(credit.Name), query)
			}
			if seen[credit.ID] {
				continue
			}
			seen[credit.ID] = true
			artist := artists[credit.ID]
			if artist == nil {
				artist = &HallArtist{HallCredit: credit, Counts: counts()}
				artists[credit.ID] = artist
			}
			artist.CertifiedSongs++
			artist.Counts[song.Current.Tier]++
			if artist.ReachedAt == nil || song.Current.ReachedAt.After(*artist.ReachedAt) {
				artist.ReachedAt = song.Current.ReachedAt
			}
		}
		if matches && (opts.Tier == "" || song.Current.Tier == opts.Tier) {
			result.Songs = append(result.Songs, song)
		}
	}
	result.CertifiedSongs = len(songs)
	result.CertifiedArtists = len(artists)
	if opts.View != "songs" {
		for _, artist := range artists {
			if strings.Contains(strings.ToLower(artist.Name), query) && (opts.Tier == "" || artist.Counts[opts.Tier] > 0) {
				result.Artists = append(result.Artists, *artist)
			}
		}
	}
	// A stable ID tie-breaker keeps page boundaries deterministic.
	less := func(cmp int, a, b int32) bool {
		if cmp == 0 {
			return a < b
		}
		if opts.Order == "asc" {
			return cmp < 0
		}
		return cmp > 0
	}
	sort.Slice(result.Songs, func(i, j int) bool {
		a, b := result.Songs[i], result.Songs[j]
		valueA, valueB := a.Plays, b.Plays
		if opts.Metric == CertificationTime {
			valueA, valueB = a.Seconds, b.Seconds
		}
		cmp := cmp.Compare(valueA, valueB)
		if opts.Sort == "name" {
			cmp = strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
		}
		if opts.Sort == "date" {
			cmp = a.Current.ReachedAt.Compare(*b.Current.ReachedAt)
		}
		return less(cmp, a.TrackID, b.TrackID)
	})
	sort.Slice(result.Artists, func(i, j int) bool {
		a, b := result.Artists[i], result.Artists[j]
		ac, bc := a.CertifiedSongs, b.CertifiedSongs
		if opts.Tier != "" {
			ac, bc = a.Counts[opts.Tier], b.Counts[opts.Tier]
		}
		cmp := cmp.Compare(ac, bc)
		if opts.Sort == "name" {
			cmp = strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		}
		if opts.Sort == "date" {
			cmp = a.ReachedAt.Compare(*b.ReachedAt)
		}
		return less(cmp, a.ID, b.ID)
	})
	paginate := func(total int) (int, int) {
		result.TotalRecordCount = total
		last := max(1, (total+result.ItemsPerPage-1)/result.ItemsPerPage)
		result.CurrentPage = min(result.CurrentPage, last)
		start := (result.CurrentPage - 1) * result.ItemsPerPage
		end := min(start+result.ItemsPerPage, total)
		result.HasNextPage = end < total
		return start, end
	}
	if opts.View == "songs" {
		start, end := paginate(len(result.Songs))
		result.Songs = result.Songs[start:end]
	} else {
		start, end := paginate(len(result.Artists))
		result.Artists = result.Artists[start:end]
	}
	return result
}
