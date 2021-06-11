package api

import (
	"errors"

	"github.com/rs/zerolog/log"
)

func HandleLogError(packageName, interfaceName, funcName string, err error, ignoredErrors ...error) {
	if err != nil {
		for _, e := range ignoredErrors {
			errors.Is(err, e)
			return
		}
		log.Debug().Err(err).Msgf("%v.%v.%v decorator intercepted error", packageName, interfaceName, funcName)
	}
}
