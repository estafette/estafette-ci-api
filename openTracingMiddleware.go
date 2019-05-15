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

		tracer := opentracing.GlobalTracer()

		span := tracer.StartSpan(fmt.Sprintf("HTTP %v %v", c.Request.Method, c.Request.URL.Path))
		defer span.Finish()

		// process request
		c.Next()
	}
}
