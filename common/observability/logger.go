package observability

// Logger is the lightweight logging contract shared across config-service packages.
// Client implements this interface, allowing one dependency for logs and events.
type Logger interface {
	Printf(format string, args ...any)
	Println(args ...any)
}

type DiscardLogger struct{}

func (DiscardLogger) Printf(string, ...any) {}

func (DiscardLogger) Println(...any) {}

func (DiscardLogger) FatalF(string, ...any) {}

func NormalizeLogger(logger Logger) Logger {
	if logger == nil {
		return DiscardLogger{}
	}
	return logger
}
