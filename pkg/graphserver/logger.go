package graphserver

// Logger is a basic levelled logger
type Logger interface {
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Log level constants
const (
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
)

// Level represents a log level
type Level int

func (l Level) String() string {
	switch l {
	case LevelError:
		return "error"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	}

	return "unknown"
}
