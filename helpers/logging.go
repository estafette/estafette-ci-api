package helpers

import (
	"errors"

	"github.com/rs/zerolog/log"
)

func HandleLogError(client string, funcName string, err error) {
	if err != nil {
		log.Error().Str("client", client).Str("func", funcName).Err(err).Msg("Failure")
	}
}

func HandleLogErrorWithIgnoredErrors(client string, funcName string, err error, ignoredErrors ...error) {
	if err != nil {
		for _, e := range ignoredErrors {
			errors.Is(err, e)
			return
		}
		log.Error().Str("client", client).Str("func", funcName).Err(err).Msg("Failure")
	}
}
