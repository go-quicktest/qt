/// Package qt implements assertion and other helpers wrapped
// around the standard library's testing types.
package qt

import (
	"testing"
)

// Assert checks that the provided argument passes the given check
// and calls tb.Error otherwise, including any Comment arguments
// in the failure.
func Assert[T any](t testing.TB, got T, checker Checker[T], comment ...Comment) bool {
	return check(t, checkParams[T]{
		fail:    t.Fatal,
		checker: checker,
		got:     got,
		comment: comment,
	})
}

// Check checks that the provided argument passes the given check
// and calls tb.Fatal otherwise, including any Comment arguments
// in the failure.
func Check[T any](t testing.TB, got T, checker Checker[T], comment ...Comment) bool {
	return check(t, checkParams[T]{
		fail:    t.Error,
		checker: checker,
		got:     got,
		comment: comment,
	})
}

func check[T any](t testing.TB, p checkParams[T]) bool {
	t.Helper()
	rp := reportParams{
		got:    p.got,
		format: Format,
	}
	// Allow checkers to annotate messages.
	note := func(key string, value any) {
		rp.notes = append(rp.notes, note{
			key:   key,
			value: value,
		})
	}
	// Ensure that we have a checker.
	if p.checker == nil {
		p.fail(report(BadCheckf("nil checker provided"), rp))
		return false
	}
	rp.args = p.checker.Args()
	rp.paramNames = p.checker.ParamNames()
	// Extract a comment if it has been provided.
	if len(p.comment) > 0 {
		rp.comment = p.comment[0]
	}
	if err := p.checker.Check(p.got, note); err != nil {
		p.fail(report(err, rp))
		return false
	}
	return true
}

type checkParams[T any] struct {
	fail    func(...any)
	checker Checker[T]
	got     T
	comment []Comment
}
