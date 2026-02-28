package nats

type Options struct {
	Host     string
	Port     string
	Username string
	Password string
}

type Option func(*Options)

func WithHost(host string) Option {
	return func(o *Options) { o.Host = host }
}

func WithPort(port string) Option {
	return func(o *Options) { o.Port = port }
}

func WithUsername(username string) Option {
	return func(o *Options) { o.Username = username }
}

func WithPassword(password string) Option {
	return func(o *Options) { o.Password = password }
}
