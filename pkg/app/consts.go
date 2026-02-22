package app

type Environment string

const (
	Production  Environment = "production"
	Development Environment = "development"
	Local       Environment = "local"
)

func (e Environment) String() string {
	return string(e)
}

func (e Environment) IsValid() bool {
	return e == Production || e == Development || e == Local
}

type LogLevel string

const (
	Debug LogLevel = "DEBUG"
	Info  LogLevel = "INFO"
	Warn  LogLevel = "WARN"
	Error LogLevel = "ERROR"
)

func (l LogLevel) String() string {
	return string(l)
}

func (l LogLevel) IsValid() bool {
	return l == Debug || l == Info || l == Warn || l == Error
}
