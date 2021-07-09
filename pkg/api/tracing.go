package api

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

func GetSpanName(prefix, funcName string) string {
	return prefix + ":" + funcName
}

func FinishSpan(span opentracing.Span) {
	span.Finish()
}

func FinishSpanWithError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	FinishSpan(span)
}
