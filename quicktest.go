// Package qt implements assertions and other helpers wrapped
// around the standard library's testing types.
package qt

import (
	"testing"
)

// Assert checks that the provided argument passes the given check
// and calls tb.Error otherwise, including any Comment arguments
// in the failure.
func Assert(t testing.TB, checker Checker, comment ...Comment) bool {
	return check(t, checkParams{
		fail:    t.Fatal,
		checker: checker,
		comment: comment,
	})
}

// Check checks that the provided argument passes the given check
// and calls tb.Fatal otherwise, including any Comment arguments
// in the failure.
func Check(t testing.TB, checker Checker, comment ...Comment) bool {
	return check(t, checkParams{
		fail:    t.Error,
		checker: checker,
		comment: comment,
	})
}

func check(t testing.TB, p checkParams) bool {
	t.Helper()
	var rp reportParams
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
	// Extract a comment if it has been provided.
	if len(p.comment) > 0 {
		rp.comment = p.comment[0]
	}
	if err := p.checker.Check(note); err != nil {
		p.fail(report(err, rp))
		return false
	}
	return true
}

type checkParams struct {
	fail    func(...any)
	checker Checker
	comment []Comment
}
