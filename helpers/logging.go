package helpers

import (
	"github.com/rs/zerolog/log"
)

func HandleLogError(client string, funcName string, err error) {
	if err != nil {
		log.Error().Str("client", client).Str("func", funcName).Err(err).Msg("Failure")
	}
}
