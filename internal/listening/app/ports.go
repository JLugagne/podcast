package app

import (
	"github.com/JLugagne/podcast/internal/listening/domain/usecases/progressuc"
)

var (
	_ progressuc.ProgressCommands = (*App)(nil)
	_ progressuc.ProgressQueries  = (*App)(nil)
)
