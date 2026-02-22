package redis

import "time"

type Options func(*options)

type options struct {
	Address      string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DefaultTTL   time.Duration
	DialTimeout  time.Duration
	KeyPrefix    string
}

func WithAddress(address string) Options {
	return func(o *options) { o.Address = address }
}

func WithPassword(password string) Options {
	return func(o *options) { o.Password = password }
}

func WithDB(db int) Options {
	return func(o *options) { o.DB = db }
}

func WithPoolSize(poolSize int) Options {
	return func(o *options) { o.PoolSize = poolSize }
}

func WithMinIdleConns(minIdleConns int) Options {
	return func(o *options) { o.MinIdleConns = minIdleConns }
}

func WithDefaultTTL(defaultTTL time.Duration) Options {
	return func(o *options) { o.DefaultTTL = defaultTTL }
}

func WithDialTimeout(dialTimeout time.Duration) Options {
	return func(o *options) { o.DialTimeout = dialTimeout }
}

func WithKeyPrefix(keyPrefix string) Options {
	return func(o *options) { o.KeyPrefix = keyPrefix }
}
