// Package converters maps listening domain values to their public HTTP
// representation.
package converters

import (
	"time"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/catalogport"
	"github.com/JLugagne/podcast/internal/listening/domain/usecases/progressuc"
	v1 "github.com/JLugagne/podcast/pkg/listening/http/v1"
)

// seriesCoverURLPrefix is the public path of a series cover.
const seriesCoverURLPrefix = "/api/v1/series/"

// ToProgress maps a progress aggregate to its public representation.
func ToProgress(p *domain.Progress) v1.Progress {
	return v1.Progress{
		EpisodeID:       string(p.EpisodeRef()),
		PositionSeconds: p.PositionSeconds(),
		Completed:       p.Completed(),
		UpdatedAt:       ts(p.UpdatedAt()),
	}
}

// ToProgressList maps progress aggregates to their public representation. The
// result is never nil so the JSON body always carries a list.
func ToProgressList(rows []*domain.Progress) v1.ProgressList {
	list := make([]v1.Progress, 0, len(rows))
	for _, row := range rows {
		list = append(list, ToProgress(row))
	}
	return v1.ProgressList{Progress: list}
}

// ToEpisodeSummary maps a catalog episode snapshot to its public
// representation.
func ToEpisodeSummary(info catalogport.EpisodeInfo) v1.EpisodeSummary {
	out := v1.EpisodeSummary{
		ID:              string(info.Ref),
		Number:          info.Number,
		Title:           info.Title,
		DurationSeconds: info.DurationSeconds,
		ChapterCount:    info.ChapterCount,
		SeriesID:        info.SeriesID,
		SeriesTitle:     info.SeriesTitle,
		SeriesSlug:      info.SeriesSlug,
		AccentColor:     info.SeriesAccentColor,
	}
	if info.SeriesHasCover {
		url := seriesCoverURLPrefix + info.SeriesID + "/cover"
		out.CoverURL = &url
	}
	return out
}

// ToContinueItem maps a continue-listening item to its public representation.
func ToContinueItem(item progressuc.ContinueItem) v1.ContinueItem {
	return v1.ContinueItem{
		Progress: ToProgress(item.Progress),
		Episode:  ToEpisodeSummary(item.Episode),
	}
}

// ToContinueList maps continue-listening items to their public representation.
// The result is never nil so the JSON body always carries a list.
func ToContinueList(items []progressuc.ContinueItem) v1.ContinueList {
	list := make([]v1.ContinueItem, 0, len(items))
	for _, item := range items {
		list = append(list, ToContinueItem(item))
	}
	return v1.ContinueList{Items: list}
}

// ts formats a timestamp as an RFC3339 string in UTC.
func ts(t time.Time) string { return t.UTC().Format(time.RFC3339) }
