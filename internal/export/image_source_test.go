package export

import "testing"

func TestExportImageSource(t *testing.T) {
	for source, want := range map[string]string{
		"https://covers.example/art.jpg": "https://covers.example/art.jpg",
		"subsonic:abc":                   "subsonic:abc",
		"http://old.example/rest/getCoverArt?id=abc&u=test&t=private": "subsonic:abc",
		"http://old.example/artist.jpg?api_key=private":               "",
	} {
		if got := exportImageSource(source); got != want {
			t.Errorf("unexpected exported source %q", got)
		}
	}
}
