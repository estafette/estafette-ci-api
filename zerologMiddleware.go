package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ZeroLogMiddleware logs gin requests via zerolog
func ZeroLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// start timer
		start := time.Now()

		// process request
		c.Next()

		// stop timer
		end := time.Now()

		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		latency := end.Sub(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		if statusCode >= 500 {

			log.Warn().
				Int("statusCode", statusCode).
				Dur("latencyMs", latency).
				Str("clientIP", clientIP).
				Str("path", path).
				Msgf("[GIN] %3d %13v %15s %-7s %s", statusCode, latency, clientIP, method, path)

		} else {

			log.Debug().
				Int("statusCode", statusCode).
				Dur("latencyMs", latency).
				Str("clientIP", clientIP).
				Str("path", path).
				Msgf("[GIN] %3d %13v %15s %-7s %s", statusCode, latency, clientIP, method, path)

		}
	}
}
