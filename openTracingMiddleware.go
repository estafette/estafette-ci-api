package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/rs/zerolog/log"
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

		// retrieve span context from upstream caller if available
		tracingCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		if err != nil {
			log.Warn().Err(err).Msgf("Failed extracting trace context from http headers for %v %v", c.Request.Method, c.Request.URL.Path)
		}

		// create a span for the http request
		span := opentracing.StartSpan(fmt.Sprintf("%v %v", c.Request.Method, c.Request.URL.Path), ext.RPCServerOption(tracingCtx))
		defer span.Finish()

		ext.SpanKindRPCServer.Set(span)
		ext.HTTPMethod.Set(span, c.Request.Method)
		ext.HTTPUrl.Set(span, c.Request.URL.String())

		// store the span in the request context
		c.Request = c.Request.WithContext(opentracing.ContextWithSpan(c.Request.Context(), span))

		// process request
		c.Next()
	}
}
