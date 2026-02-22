package otlp

import "github.com/karavanix/karavantrack-api-server/pkg/app"

type ExporterType int

const (
	ExporterOTLP ExporterType = iota
	ExporterStdout
)

var ExporterNameToExporterType = map[string]ExporterType{
	"otlp":   ExporterOTLP,
	"stdout": ExporterStdout,
}

type ExporterProtocolType string

const (
	ExporterProtocolHTTP ExporterProtocolType = "http"
	ExporterProtocolGRPC ExporterProtocolType = "grpc"
)

var ExporterProtocolNameToExporterProtocolType = map[string]ExporterProtocolType{
	"http": ExporterProtocolHTTP,
	"grpc": ExporterProtocolGRPC,
}

type SamplerType string

const (
	AlwaysOn                SamplerType = "always_on"
	AlwaysOff               SamplerType = "always_off"
	TraceIDRatio            SamplerType = "traceidratio"
	ParentBasedAlwaysOn     SamplerType = "parentbased_always_on"
	ParentBasedAlwaysOff    SamplerType = "parentbased_always_off"
	ParentBasedTraceIDRatio SamplerType = "parentbased_traceidratio"
)

var SamplerNameToSamplerType = map[string]SamplerType{
	"always_on":                AlwaysOn,
	"always_off":               AlwaysOff,
	"traceidratio":             TraceIDRatio,
	"parentbased_always_on":    ParentBasedAlwaysOn,
	"parentbased_always_off":   ParentBasedAlwaysOff,
	"parentbased_traceidratio": ParentBasedTraceIDRatio,
}

type Option func(*Options)

type Options struct {
	ServiceName      string
	Endpoint         string
	Environment      app.Environment
	SamplerArg       string // string in env, parsed later
	SamplerType      SamplerType
	ExporterType     ExporterType
	ExporterProtocol ExporterProtocolType
	Propagators      []string // e.g. ["baggage", "tracecontext"]
}

func WithServiceName(serviceName string) Option {
	return func(o *Options) { o.ServiceName = serviceName }
}
func WithEndpoint(endpoint string) Option {
	return func(o *Options) { o.Endpoint = endpoint }
}
func WithEnvironment(environment app.Environment) Option {
	return func(o *Options) { o.Environment = environment }
}
func WithSamplerArg(samplerArg string) Option {
	return func(o *Options) { o.SamplerArg = samplerArg }
}
func WithSamplerType(samplerType SamplerType) Option {
	return func(o *Options) { o.SamplerType = samplerType }
}
func WithExporterType(exporterType ExporterType) Option {
	return func(o *Options) { o.ExporterType = exporterType }
}
func WithExporterProtocol(p ExporterProtocolType) Option {
	return func(o *Options) { o.ExporterProtocol = p }
}
func WithPropagators(props []string) Option {
	return func(o *Options) { o.Propagators = props }
}
