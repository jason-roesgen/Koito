package export

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/models"
	"github.com/google/uuid"
)

type KoitoExport struct {
	Version    string        `json:"version"`
	ExportedAt time.Time     `json:"exported_at"` // RFC3339
	User       string        `json:"user"`        // username
	Listens    []KoitoListen `json:"listens"`
}
type KoitoListen struct {
	ListenedAt time.Time     `json:"listened_at"`
	Client     string        `json:"client"`
	Track      KoitoTrack    `json:"track"`
	Album      KoitoAlbum    `json:"album"`
	Artists    []KoitoArtist `json:"artists"`
}
type KoitoTrack struct {
	MBID     *uuid.UUID     `json:"mbid"`
	Duration int            `json:"duration"`
	Aliases  []models.Alias `json:"aliases"`
}
type KoitoAlbum struct {
	ImageUrl       string         `json:"image_url"`
	MBID           *uuid.UUID     `json:"mbid"`
	Aliases        []models.Alias `json:"aliases"`
	VariousArtists bool           `json:"various_artists"`
}
type KoitoArtist struct {
	ImageUrl  string         `json:"image_url"`
	MBID      *uuid.UUID     `json:"mbid"`
	IsPrimary bool           `json:"is_primary"`
	Aliases   []models.Alias `json:"aliases"`
}

func ExportData(ctx context.Context, user *models.User, store db.ExportStore, out io.Writer) error {
	lastTime := time.Unix(0, 0)
	lastTrackId := int32(0)
	pageSize := int32(1000)

	l := logger.FromContext(ctx)
	l.Info().Msg("ExportData: Generating Koito export file...")

	exportedAt := time.Now()
	// exportFile := path.Join(cfg.ConfigDir(), fmt.Sprintf("koito_export_%d.json", exportedAt.Unix()))
	// f, err := os.Create(exportFile)
	// if err != nil {
	// 	return fmt.Errorf("ExportData: %w", err)
	// }
	// defer f.Close()

	// Write the opening of the JSON manually
	username, err := json.Marshal(user.Username)
	if err != nil {
		return fmt.Errorf("ExportData: marshal username: %w", err)
	}
	_, err = fmt.Fprintf(out, "{\n  \"version\": \"1\",\n  \"exported_at\": \"%s\",\n  \"user\": %s,\n  \"listens\": [\n", exportedAt.UTC().Format(time.RFC3339), username)
	if err != nil {
		return fmt.Errorf("ExportData: %w", err)
	}

	first := true
	for {
		rows, err := store.GetExportPage(ctx, db.GetExportPageOpts{
			UserID:     user.ID,
			ListenedAt: lastTime,
			TrackID:    lastTrackId,
			Limit:      pageSize,
		})
		if err != nil {
			return fmt.Errorf("ExportData: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		for _, r := range rows {
			// Adds a comma after each listen item
			if !first {
				if _, err := io.WriteString(out, ",\n"); err != nil {
					return fmt.Errorf("ExportData: %w", err)
				}
			}
			first = false

			exported := convertToExportFormat(r)

			raw, err := json.MarshalIndent(exported, "    ", "  ")

			if err != nil {
				return fmt.Errorf("ExportData: marshal: %w", err)
			}
			if _, err := out.Write(append([]byte("    "), raw...)); err != nil {
				return fmt.Errorf("ExportData: %w", err)
			}
		}
		last := rows[len(rows)-1]
		lastTime, lastTrackId = last.ListenedAt, last.TrackID
	}

	// Write closing of the JSON array and object
	_, err = out.Write([]byte("\n  ]\n}\n"))
	if err != nil {
		return fmt.Errorf("ExportData: f.Write: %w", err)
	}

	l.Info().Msgf("Export successfully created")
	return nil
}

func convertToExportFormat(item *db.ExportItem) *KoitoListen {
	var client string
	if item.Client != nil {
		client = *item.Client
	}
	ret := &KoitoListen{
		ListenedAt: item.ListenedAt.UTC(),
		Track: KoitoTrack{
			MBID:     item.TrackMbid,
			Duration: int(item.TrackDuration),
			Aliases:  item.TrackAliases,
		},
		Client: client,
		Album: KoitoAlbum{
			MBID:           item.ReleaseMbid,
			ImageUrl:       exportImageSource(item.ReleaseImageSource),
			VariousArtists: item.VariousArtists,
			Aliases:        item.ReleaseAliases,
		},
	}
	for i := range item.Artists {
		ret.Artists = append(ret.Artists, KoitoArtist{
			IsPrimary: item.Artists[i].IsPrimary,
			MBID:      item.Artists[i].MbzID,
			Aliases:   item.Artists[i].Aliases,
			ImageUrl:  exportImageSource(item.Artists[i].ImageSource),
		})
	}
	return ret
}

func exportImageSource(source string) string {
	u, err := url.Parse(source)
	if err != nil || u.User != nil {
		return ""
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		if strings.HasSuffix(strings.TrimSuffix(u.Path, ".view"), "/rest/getCoverArt") {
			if id := u.Query().Get("id"); id != "" {
				return "subsonic:" + id
			}
			return ""
		}
		for _, key := range []string{"u", "p", "t", "s", "api_key", "token"} {
			if u.Query().Has(key) {
				return ""
			}
		}
	}
	return source
}
