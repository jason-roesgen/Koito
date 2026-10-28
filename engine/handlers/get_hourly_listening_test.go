package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
)

type hourlyStoreStub struct {
	called    bool
	timeframe db.Timeframe
	err       error
}

func (s *hourlyStoreStub) GetHourlyListening(_ context.Context, timeframe db.Timeframe) ([]db.HourlyListeningBucket, error) {
	s.called = true
	s.timeframe = timeframe
	return []db.HourlyListeningBucket{{Hour: 7, Plays: 2, Seconds: 300}}, s.err
}

func TestHourlyListeningHandler(t *testing.T) {
	if err := cfg.Load(func(string) string { return "" }, "test"); err != nil {
		t.Fatal(err)
	}
	t.Run("timezone and period", func(t *testing.T) {
		store := &hourlyStoreStub{}
		response := httptest.NewRecorder()
		GetHourlyListeningHandler(store).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/?period=week&tz=Asia/Kathmandu", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
		if store.timeframe.Period != db.PeriodWeek || store.timeframe.Timezone.String() != "Asia/Kathmandu" {
			t.Fatal("incorrect query options")
		}
		var result struct {
			Timezone string                     `json:"timezone"`
			Hours    []db.HourlyListeningBucket `json:"hours"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Timezone != "Asia/Kathmandu" || len(result.Hours) != 1 || result.Hours[0].Seconds != 300 {
			t.Fatal("incorrect response")
		}
	})
	t.Run("invalid period", func(t *testing.T) {
		for _, period := range []string{"", "invalid"} {
			store := &hourlyStoreStub{}
			response := httptest.NewRecorder()
			GetHourlyListeningHandler(store).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/?period="+period, nil))
			if response.Code != http.StatusBadRequest || store.called {
				t.Fatalf("invalid period %q was not rejected", period)
			}
		}
	})
	t.Run("store error", func(t *testing.T) {
		store := &hourlyStoreStub{err: errors.New("database unavailable")}
		response := httptest.NewRecorder()
		GetHourlyListeningHandler(store).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/?period=all_time&tz=UTC", nil))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status %d", response.Code)
		}
	})
}
