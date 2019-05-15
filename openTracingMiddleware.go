package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
)

// OpenTracingMiddleware creates a span for each request
func OpenTracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		path := c.Request.URL.Path
		if path == "/liveness" || path == "/readiness" {
			// don't log these requests, only execute them
			c.Next()
			return
		}

		// create a span for the http request
		span := opentracing.StartSpan(fmt.Sprintf("HTTP %v %v", c.Request.Method, c.Request.URL.Path))
		defer span.Finish()

		// store the span in the request context
		c.Request = c.Request.WithContext(opentracing.ContextWithSpan(c.Request.Context(), span))

		// process request
		c.Next()
	}
}
