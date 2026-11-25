package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gabehf/koito/internal/cfg"
)

func TestRequestPaginationBounds(t *testing.T) {
	if err := cfg.Load(func(string) string { return "" }, "test"); err != nil {
		t.Fatal(err)
	}
	for input, want := range map[string]int{"": 100, "0": 100, "-1": 100, "1": 1, "500": 500, "501": 100} {
		opts := OptsFromRequest(httptest.NewRequest("GET", "/?page=0&limit="+input, nil))
		if opts.Limit != want || opts.Page != 1 {
			t.Errorf("limit=%q returned limit %d, page %d", input, opts.Limit, opts.Page)
		}
	}
}
