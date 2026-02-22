package postgres

type Options func(*options)

type options struct {
	Host               string
	Port               string
	User               string
	Password           string
	DB                 string
	Debug              bool
	SLLMode            string
	MaxOpenConnections int
	MaxIdleConnections int
	ConnectTimeout     string
}

func WithHost(host string) Options {
	return func(o *options) { o.Host = host }
}

func WithPort(port string) Options {
	return func(o *options) { o.Port = port }
}

func WithUser(user string) Options {
	return func(o *options) { o.User = user }
}

func WithPassword(password string) Options {
	return func(o *options) { o.Password = password }
}

func WithDB(db string) Options {
	return func(o *options) { o.DB = db }
}

func WithSSLMode(sslMode string) Options {
	return func(o *options) { o.SLLMode = sslMode }
}

func WithMaxOpenConnections(maxOpenConnections int) Options {
	return func(o *options) { o.MaxOpenConnections = maxOpenConnections }
}

func WithMaxIdleConnections(maxIdleConnections int) Options {
	return func(o *options) { o.MaxIdleConnections = maxIdleConnections }
}

func WithConnectTimeout(connectTimeout string) Options {
	return func(o *options) { o.ConnectTimeout = connectTimeout }
}

func WithDebug(debug bool) Options {
	return func(o *options) { o.Debug = debug }
}
