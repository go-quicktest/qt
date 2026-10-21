// Package qt implements assertions and other helpers wrapped around the
// standard library's testing types.
package qt

import (
	"testing"
)

// Assert checks that the provided argument passes the given check and calls
// tb.Fatal otherwise, including any Comment arguments in the failure.
func Assert(t testing.TB, checker Checker, comments ...Comment) bool {
	if msg, ok := check(checker, comments); !ok {
		t.Helper()
		t.Fatal(msg)
		return false
	}
	return true
}

// Check checks that the provided argument passes the given check and calls
// tb.Error otherwise, including any Comment arguments in the failure.
func Check(t testing.TB, checker Checker, comments ...Comment) bool {
	if msg, ok := check(checker, comments); !ok {
		t.Helper()
		t.Error(msg)
		return false
	}
	return true
}

// check runs the checker, returning a failure report if it fails.
func check(checker Checker, comments []Comment) (string, bool) {
	// Ensure that we have a checker.
	if checker == nil {
		return report(BadCheckf("nil checker provided"), reportParams{
			comments: comments,
		}), false
	}

	// Allow checkers to annotate messages.
	var notes []note
	note := func(key string, value any) {
		notes = append(notes, note{
			key:   key,
			value: value,
		})
	}

	// Run the check.
	if err := checker.Check(note); err != nil {
		return report(err, reportParams{
			checker:  checker,
			comments: comments,
			notes:    notes,
		}), false
	}
	return "", true
}
