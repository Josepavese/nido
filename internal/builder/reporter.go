package builder

import "io"

type Reporter interface {
	Header(title string)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Success(msg string, args ...any)
}

type NopReporter struct{}

func (NopReporter) Header(string)          {}
func (NopReporter) Info(string, ...any)    {}
func (NopReporter) Warn(string, ...any)    {}
func (NopReporter) Success(string, ...any) {}

type EngineOption func(*Engine)

func WithReporter(r Reporter) EngineOption {
	return func(e *Engine) {
		if r != nil {
			e.Reporter = r
		}
	}
}

func WithCommandOutput(w io.Writer) EngineOption {
	return func(e *Engine) {
		e.CommandOutput = w
	}
}
