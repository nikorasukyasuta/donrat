package utils

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// NewLogger creates a structured zerolog logger.
func NewLogger(level string, env string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	parsedLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		parsedLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(parsedLevel)

	logger := zerolog.New(os.Stdout).With().Timestamp().Str("env", env).Logger()
	return logger
}
