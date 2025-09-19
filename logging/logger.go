package logging

import (
	"io"
	"os"

	axiomAdapter "github.com/axiomhq/axiom-go/adapters/zerolog"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Setup configures zerolog with appropriate formatting
func Setup(dev bool) {
	// set global log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if dev {
		// pretty printing for development
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
			With().Timestamp().Caller().Logger()
	} else {
		// json format for production
		if os.Getenv("AXIOM_TOKEN") == "" {
			log.Logger = zerolog.New(os.Stderr).With().Caller().Logger()
			log.Warn().Msg("Axiom token not set, logging to stderr only")
		} else {
			writer, err := axiomAdapter.New(
				axiomAdapter.SetDataset(os.Getenv("AXIOM_DATASET")),
			)
			if err != nil {
				log.Fatal().Err(err).Msg("Error initializing Axiom adapter")
			}
			log.Logger = zerolog.New(io.MultiWriter(os.Stderr, writer)).With().Caller().Timestamp().Logger()
		}
	}
}

// Info logs an info message
func Info(message string, fields ...interface{}) {
	log.Info().Fields(fields).Msg(message)
}

// Error logs an error message
func Error(message string, err error, fields ...interface{}) {
	log.Error().Err(err).Fields(fields).Msg(message)
}

// Debug logs a debug message
func Debug(message string, fields ...interface{}) {
	log.Debug().Fields(fields).Msg(message)
}

// Warn logs a warning message
func Warn(message string, fields ...interface{}) {
	log.Warn().Fields(fields).Msg(message)
}
