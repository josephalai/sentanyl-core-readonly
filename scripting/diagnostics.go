package scripting

import "fmt"

// DiagLevel indicates severity.
type DiagLevel int

const (
	DiagError DiagLevel = iota
	DiagWarning
	DiagInfo
)

func (d DiagLevel) String() string {
	switch d {
	case DiagError:
		return "error"
	case DiagWarning:
		return "warning"
	case DiagInfo:
		return "info"
	}
	return "unknown"
}

// Diagnostic is a single error/warning with source location.
type Diagnostic struct {
	Pos     Pos
	Message string
	Level   DiagLevel
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("[%s] %s: %s", d.Level, d.Pos, d.Message)
}

// Diagnostics is a collection of diagnostics.
type Diagnostics []Diagnostic

func (ds Diagnostics) HasErrors() bool {
	for _, d := range ds {
		if d.Level == DiagError {
			return true
		}
	}
	return false
}

func (ds Diagnostics) Errors() Diagnostics {
	var out Diagnostics
	for _, d := range ds {
		if d.Level == DiagError {
			out = append(out, d)
		}
	}
	return out
}
