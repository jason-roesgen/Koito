package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabehf/koito/internal/db"
	"github.com/go-chi/chi/v5"
)

type certificationStub struct {
	id     int32
	metric string
	err    error
}

func (s *certificationStub) GetTrackCertifications(_ context.Context, id int32, metric string) (*db.TrackCertifications, error) {
	s.id, s.metric = id, metric
	return db.NewTrackCertifications(id, "test"), s.err
}
func (s *certificationStub) GetArtistCertifications(_ context.Context, id int32, metric string) (*db.ArtistCertifications, error) {
	s.id, s.metric = id, metric
	return &db.ArtistCertifications{Counts: map[string]int{}, Tracks: []*db.TrackCertifications{}}, s.err
}
func (s *certificationStub) GetMilestoneHistory(context.Context) (*db.MilestoneHistory, error) {
	return &db.MilestoneHistory{}, s.err
}
func TestCertificationHandler(t *testing.T) {
	for _, kind := range []string{"track", "artist"} {
		for _, tt := range []struct {
			id     string
			err    error
			status int
		}{
			{"1", nil, 200}, {"1", db.ErrNotFound, 404}, {"1", errors.New("database error"), 500},
			{"0", nil, 400}, {"-1", nil, 400}, {"abc", nil, 400}, {"2147483648", nil, 400},
		} {
			store := &certificationStub{err: tt.err}
			router := chi.NewRouter()
			router.Get("/{id}", GetCertificationsHandler(store, kind))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/"+tt.id, nil))
			if response.Code != tt.status {
				t.Fatalf("%s %s: got %d want %d", kind, tt.id, response.Code, tt.status)
			}
			if tt.status == 400 && store.id != 0 {
				t.Fatal("invalid id reached store")
			}
		}
	}
	response := httptest.NewRecorder()
	GetCertificationsHandler(&certificationStub{}, "milestones").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != 200 {
		t.Fatalf("milestones: %d", response.Code)
	}
}

func TestCertificationMetricSelection(t *testing.T) {
	for _, kind := range []string{"track", "artist"} {
		for _, metric := range []string{"time", "unknown"} {
			store := &certificationStub{}
			router := chi.NewRouter()
			router.Get("/{id}", GetCertificationsHandler(store, kind))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/1?metric="+metric, nil))
			if metric == "time" && (w.Code != http.StatusOK || store.metric != db.CertificationTime) {
				t.Fatalf("%s time mode: status %d, metric %q", kind, w.Code, store.metric)
			}
			if metric == "unknown" && (w.Code != http.StatusBadRequest || store.id != 0) {
				t.Fatalf("%s invalid mode reached store: status %d, id %d", kind, w.Code, store.id)
			}
		}
	}
}
