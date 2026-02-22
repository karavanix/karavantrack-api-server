package otlp

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Start creates a span and returns ctx + a finisher that handles error/status.
func Start(ctx context.Context, tracer trace.Tracer, name string, attrs ...attribute.KeyValue) (context.Context, func(err error)) {
	ctx, span := tracer.Start(ctx, name, trace.WithAttributes(attrs...))
	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

// Annotate adds attributes to the current span if present.
func Annotate(ctx context.Context, attrs ...attribute.KeyValue) {
	if span := trace.SpanFromContext(ctx); span != nil {
		span.SetAttributes(attrs...)
	}
}

// Event adds a breadcrumb to the current span.
func Event(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	if span := trace.SpanFromContext(ctx); span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RestoreTraceContext function forms context and span from trace_id and span_id
//
// span_id and trace_id should both be strings in hex format.
//
// Although this function returns both context and span it is required to call Start method to start tracing
// WARNING: if error IS NOT NIL, then context and span are BOTH NIL.
func RestoreTraceContext(traceIdStr, spanIdStr string) (context.Context, trace.Span, error) {
	spanId, err := trace.SpanIDFromHex(spanIdStr)
	if err != nil {
		return nil, nil, err
	}

	traceId, err := trace.TraceIDFromHex(traceIdStr)
	if err != nil {
		return nil, nil, err
	}

	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,
	}))

	return ctx, trace.SpanFromContext(ctx), nil
}
