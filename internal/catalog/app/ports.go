package app

import (
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
)

var (
	_ seriesuc.SeriesCommands   = (*App)(nil)
	_ seriesuc.SeriesQueries    = (*App)(nil)
	_ episodeuc.EpisodeCommands = (*App)(nil)
	_ episodeuc.EpisodeQueries  = (*App)(nil)
)
