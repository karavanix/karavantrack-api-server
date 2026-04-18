package s3

type Options struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Secure    bool
}

type Option func(*Options)

func WithEndpoint(endpoint string) Option {
	return func(o *Options) {
		o.Endpoint = endpoint
	}
}

func WithRegion(region string) Option {
	return func(o *Options) {
		o.Region = region
	}
}

func WithAccessKey(accessKey string) Option {
	return func(o *Options) {
		o.AccessKey = accessKey
	}
}

func WithSecretKey(secretKey string) Option {
	return func(o *Options) {
		o.SecretKey = secretKey
	}
}

func WithSecure(secure bool) Option {
	return func(o *Options) {
		o.Secure = secure
	}
}
