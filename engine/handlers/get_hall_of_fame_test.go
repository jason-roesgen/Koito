package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/db/sqlite"
)

type hallStub struct {
	opts  db.HallOfFameOpts
	calls int
	err   error
}

func (s *hallStub) GetHallOfFame(_ context.Context, opts db.HallOfFameOpts) (*db.HallOfFame, error) {
	s.opts = opts
	s.calls++
	return db.BuildHallOfFame(nil, opts), s.err
}
func TestHallOfFameHandler(t *testing.T) {
	for _, query := range []string{"view=albums", "tier=unknown", "sort=plays", "order=up", "page=0", "page=-1", "page=abc", "page=1000001", "page=99999999999999999999999", "q=" + strings.Repeat("x", 201)} {
		s := &hallStub{}
		w := httptest.NewRecorder()
		GetHallOfFameHandler(s)(w, httptest.NewRequest("GET", "/?"+query, nil))
		if w.Code != 400 || s.calls != 0 || !json.Valid(w.Body.Bytes()) {
			t.Fatalf("invalid filters accepted: %s %d", query, w.Code)
		}
	}
	s := &hallStub{}
	w := httptest.NewRecorder()
	GetHallOfFameHandler(s)(w, httptest.NewRequest("GET", "/", nil))
	var response db.HallOfFame
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &response) != nil || response.Songs == nil || response.Artists == nil || s.opts.View != "artists" || s.opts.Page != 1 {
		t.Fatalf("bad empty HTTP response: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	GetHallOfFameHandler(s)(w, httptest.NewRequest("GET", "/?view=songs&tier=gold&sort=name&order=asc&page=2&q=%27%3BSELECT", nil))
	if w.Code != 200 || s.opts.Query != "';SELECT" || s.opts.Tier != "gold" || s.opts.Page != 2 || s.opts.Sort != "name" || s.opts.Order != "asc" {
		t.Fatalf("valid options lost: %+v", s.opts)
	}
	for _, n := range []int{200, 201} {
		term := strings.Repeat("曲", n)
		w = httptest.NewRecorder()
		GetHallOfFameHandler(s)(w, httptest.NewRequest("GET", "/?q="+url.QueryEscape(term), nil))
		if n == 200 && (w.Code != 200 || s.opts.Query != term) || n == 201 && w.Code != 400 {
			t.Fatalf("Unicode search with %d characters returned %d", n, w.Code)
		}
	}
	s.err = errors.New("private DB detail")
	w = httptest.NewRecorder()
	GetHallOfFameHandler(s)(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 500 || !json.Valid(w.Body.Bytes()) || strings.Contains(w.Body.String(), "private DB detail") {
		t.Fatal("error leaked or did not fail")
	}
}

// Exercise serialization and mutations through the real handler/store boundary.
func TestHallOfFameHTTPTransition(t *testing.T) {
	s, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	exec := func(q string, args ...any) {
		t.Helper()
		if err := s.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	request := func() *db.HallOfFame {
		t.Helper()
		w := httptest.NewRecorder()
		GetHallOfFameHandler(s)(w, httptest.NewRequest("GET", "/?view=songs", nil))
		var result db.HallOfFame
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatalf("bad HTTP: %d %s", w.Code, w.Body.String())
		}
		return &result
	}
	if request().CertifiedSongs != 0 {
		t.Fatal("not empty")
	}
	exec("INSERT INTO users VALUES(1,'test','user','test')")
	exec("INSERT INTO artists(id) VALUES(1)")
	exec("INSERT INTO artist_aliases VALUES(1,'Artist','test',1)")
	exec("INSERT INTO releases(id) VALUES(1)")
	exec("INSERT INTO tracks(id,release_id) VALUES(1,1)")
	exec("INSERT INTO track_aliases VALUES(1,'Track','test',1)")
	exec("INSERT INTO artist_tracks VALUES(1,1,1)")
	for n := 1; n <= 199; n++ {
		exec("INSERT INTO listens(track_id,listened_at,user_id) VALUES(1,?,1)", n)
	}
	if request().CertifiedSongs != 0 {
		t.Fatal("199 plays certified")
	}
	exec("INSERT INTO listens(track_id,listened_at,user_id) VALUES(1,200,1)")
	if r := request(); r.CertifiedSongs != 1 || r.Songs[0].Current.Tier != "gold" || r.Songs[0].Current.ReachedAt.Unix() != 200 {
		t.Fatalf("first Gold missing: %+v", r)
	}
	exec("DELETE FROM listens WHERE track_id=1 AND listened_at=200")
	if request().CertifiedSongs != 0 {
		t.Fatal("deleted Gold remained")
	}
}
