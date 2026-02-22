package otlp

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

var once sync.Once
var TracerProvider *trace.TracerProvider

// InitTracer keeps your original API (opts pattern).
func InitTracer(ctx context.Context, opts ...Option) func(context.Context) error {
	once.Do(func() {
		o := &Options{
			ServiceName:      "my-service",
			Endpoint:         "http://localhost:4317",
			Environment:      app.Local,
			SamplerArg:       "1.0",
			SamplerType:      AlwaysOff,
			ExporterType:     ExporterStdout,
			ExporterProtocol: ExporterProtocolGRPC,
			Propagators:      []string{"baggage", "tracecontext"},
		}
		for _, opt := range opts {
			opt(o)
		}

		res, err := resource.New(
			ctx,
			resource.WithAttributes(
				semconv.ServiceName(o.ServiceName),
				semconv.ServiceVersion("1.0.0"),
				semconv.DeploymentEnvironmentName(o.Environment.String()),
			),
		)
		if err != nil {
			log.Fatalf("failed to create resource: %v", err)
		}

		exp, err := buildExporter(ctx, o)
		if err != nil {
			log.Fatalf("failed to create exporter: %v", err)
		}

		bsp := trace.NewBatchSpanProcessor(exp)
		TracerProvider = trace.NewTracerProvider(
			trace.WithSampler(buildSampler(o.SamplerType, o.SamplerArg)),
			trace.WithResource(res),
			trace.WithSpanProcessor(bsp),
		)

		otel.SetTracerProvider(TracerProvider)
		otel.SetTextMapPropagator(buildPropagator(o.Propagators))
	})

	return TracerProvider.Shutdown
}

// --- helpers ---

func buildExporter(ctx context.Context, o *Options) (trace.SpanExporter, error) {
	switch o.ExporterType {
	case ExporterOTLP:
		switch o.ExporterProtocol {
		case ExporterProtocolHTTP:
			httpClient := otlptracehttp.NewClient(
				otlptracehttp.WithInsecure(),
				withHttpEndpoint(o.Endpoint),
			)
			return otlptrace.New(ctx, httpClient)
		default: // grpc
			grpcClient := otlptracegrpc.NewClient(
				otlptracegrpc.WithInsecure(),
				withGrpcEndpoint(o.Endpoint),
			)
			return otlptrace.New(ctx, grpcClient)
		}
	default:
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
			stdouttrace.WithoutTimestamps(),
		)
	}
}

func withGrpcEndpoint(endpoint string) otlptracegrpc.Option {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return otlptracegrpc.WithEndpointURL(endpoint)
	}

	return otlptracegrpc.WithEndpoint(endpoint)
}

func withHttpEndpoint(endpoint string) otlptracehttp.Option {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return otlptracehttp.WithEndpointURL(endpoint)
	}

	return otlptracehttp.WithEndpoint(endpoint)
}

func buildPropagator(names []string) propagation.TextMapPropagator {
	var ps []propagation.TextMapPropagator
	seen := map[string]struct{}{}

	for _, n := range names {
		name := strings.TrimSpace(strings.ToLower(n))
		if _, ok := seen[name]; ok || name == "" {
			continue
		}
		seen[name] = struct{}{}

		switch name {
		case "tracecontext":
			ps = append(ps, propagation.TraceContext{})
		case "baggage":
			ps = append(ps, propagation.Baggage{})
		case "b3":
			ps = append(ps, b3.New())
		case "b3multi":
			ps = append(ps, b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)))
		default:
			// ignore unknowns to be lenient
		}
	}

	if len(ps) == 0 {
		return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}
	return propagation.NewCompositeTextMapPropagator(ps...)
}

func buildSampler(samplerType SamplerType, arg string) trace.Sampler {
	ratio := parseRatio(arg, 1.0)

	switch samplerType {
	case AlwaysOn:
		return trace.AlwaysSample()

	case AlwaysOff:
		return trace.NeverSample()

	case TraceIDRatio:
		return trace.TraceIDRatioBased(clamp01(ratio))

	case ParentBasedAlwaysOn:
		return trace.ParentBased(trace.AlwaysSample())

	case ParentBasedAlwaysOff:
		return trace.ParentBased(trace.NeverSample())

	case ParentBasedTraceIDRatio:
		return trace.ParentBased(trace.TraceIDRatioBased(clamp01(ratio)))

	default:
		log.Printf("[otel] unknown OTEL_TRACES_SAMPLER=%q; falling back to parentbased_always_off", samplerType)
		return trace.ParentBased(trace.NeverSample())
	}
}

func parseRatio(v string, def float64) float64 {
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Printf("[otel] invalid OTEL_TRACES_SAMPLER_ARG=%q; using default=%v", v, def)
		return def
	}
	return f
}

func clamp01(f float64) float64 {
	switch {
	case f < 0:
		return 0
	case f > 1:
		return 1
	default:
		return f
	}
}
