package qt

import (
	"runtime"
	"testing"
)

// A Marker records a name at a given Go source file position.
type Marker struct {
	_    struct{} // Prevent
	Name string
	File string
	Line int
}

// Mark returns a Marker with the given name
// that also records the Go source file position where it appears.
func Mark(name string) Marker {
	_, file, line, _ := runtime.Caller(1)
	return Marker{
		Name: name,
		File: file,
		Line: line,
	}
}

// Source returns the File and Line.
func (m Marker) Source() (string, int) { return m.File, m.Line }

func (m Marker) Run(t *testing.T, f func(*testing.T)) bool {
	return t.Run(m.Name, func(t *testing.T) {
		t.Cleanup(func() {
			t.Logf("%s:%d: test location", m.File, m.Line)
		})
		f(t)
	})
}

func (m Marker) Benchmark(b *testing.B, f func(*testing.B)) bool {
	return b.Run(m.Name, func(b *testing.B) {
		b.Cleanup(func() {
			b.Logf("%s:%d: test location", m.File, m.Line)
		})
		f(b)
	})
}
