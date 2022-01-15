// Licensed under the MIT license, see LICENCE file for details.

package qt_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
)

type qtTest[T any] struct {
	about           string
	checker         qt.Checker[T]
	got             T
	comment         []qt.Comment
	expectedFailure string
}

type testRunner interface {
	run(t *testing.T)
}

var qtTests = []testRunner{qtTest[int]{
	about:   "success",
	checker: qt.Equals(42),
	got:     42,
}, qtTest[string]{
	about:   "failure",
	checker: qt.Equals("47"),
	got:     "42",
	expectedFailure: `
error:
  values are not equal
got:
  "42"
want:
  "47"
`,
}, qtTest[string]{
	about:   "failure with % signs",
	checker: qt.Equals("47%y"),
	got:     "42%x",
	expectedFailure: `
error:
  values are not equal
got:
  "42%x"
want:
  "47%y"
`,
}, qtTest[bool]{
	about:   "failure with comment",
	checker: qt.Equals(false),
	got:     true,
	comment: []qt.Comment{qt.Commentf("apparently %v != %v", true, false)},
	expectedFailure: `
error:
  values are not equal
comment:
  apparently true != false
got:
  bool(true)
want:
  bool(false)
`,
}, qtTest[any]{
	about:   "another failure with comment",
	checker: qt.IsNil,
	got:     42,
	comment: []qt.Comment{qt.Commentf("bad wolf: %d", 42)},
	expectedFailure: `
error:
  got non-nil value
comment:
  bad wolf: 42
got:
  int(42)
`,
}, qtTest[any]{
	about:   "failure with constant comment",
	checker: qt.IsNil,
	got:     "something",
	comment: []qt.Comment{qt.Commentf("these are the voyages")},
	expectedFailure: `
error:
  got non-nil value
comment:
  these are the voyages
got:
  "something"
`,
}, qtTest[any]{
	about:   "failure with empty comment",
	checker: qt.IsNil,
	got:     47,
	comment: []qt.Comment{qt.Commentf("")},
	expectedFailure: `
error:
  got non-nil value
got:
  int(47)
`,
}, qtTest[int]{
	about: "nil checker",
	expectedFailure: `
error:
  bad check: nil checker provided
`,
}, qtTest[int]{
	about: "many arguments and notes",
	checker: &testingChecker[int]{
		paramNames: []string{"arg1", "arg2", "arg3"},
		args:       []any{"val2", "val3"},
		addNotes: func(note func(key string, value any)) {
			note("note1", "these")
			note("note2", qt.Unquoted("are"))
			note("note3", "the")
			note("note4", "voyages")
			note("note5", true)
		},
		err: errors.New("bad wolf"),
	},
	got: 42,
	expectedFailure: `
error:
  bad wolf
note1:
  "these"
note2:
  are
note3:
  "the"
note4:
  "voyages"
note5:
  bool(true)
arg1:
  int(42)
arg2:
  "val2"
arg3:
  "val3"
`,
}, qtTest[string]{
	about: "many arguments and notes with the same value",
	checker: &testingChecker[string]{
		paramNames: []string{"arg1", "arg2", "arg3", "arg4"},
		args:       []any{"value1", []int{42}, nil},
		addNotes: func(note func(key string, value any)) {
			note("note1", "value1")
			note("note2", []int{42})
			note("note3", "value1")
			note("note4", nil)
		},
		err: errors.New("bad wolf"),
	},
	got: "value1",
	expectedFailure: `
error:
  bad wolf
note1:
  "value1"
note2:
  []int{42}
note3:
  <same as "note1">
note4:
  nil
arg1:
  <same as "note1">
arg2:
  <same as "note1">
arg3:
  <same as "note2">
arg4:
  <same as "note4">
`,
}, qtTest[int]{
	about: "bad check with notes",
	checker: &testingChecker[int]{
		paramNames: []string{"got", "want"},
		addNotes: func(note func(key string, value any)) {
			note("note", 42)
		},
		err:  qt.BadCheckf("bad wolf"),
		args: []any{"want"},
	},
	got: 42,
	expectedFailure: `
error:
  bad check: bad wolf
note:
  int(42)
`,
}, qtTest[int]{
	about: "silent failure with notes",
	checker: &testingChecker[int]{
		paramNames: []string{"got", "want"},
		addNotes: func(note func(key string, value any)) {
			note("note1", "first note")
			note("note2", qt.Unquoted("second note"))
		},
		args: []any{"want"},
		err:  qt.ErrSilent,
	},
	got: 42,
	expectedFailure: `
note1:
  "first note"
note2:
  second note
`,
}}

func TestCAssertCheck(t *testing.T) {
	for _, test := range qtTests {
		test.run(t)
	}
}

func (test qtTest[T]) run(t *testing.T) {
	t.Run("Assert: "+test.about, func(t *testing.T) {
		tt := &testingT{}
		ok := qt.Assert(tt, test.got, test.checker, test.comment...)
		checkResult(t, ok, tt.fatalString(), test.expectedFailure)
		if tt.errorString() != "" {
			t.Fatalf("no error messages expected, but got %q", tt.errorString())
		}
	})
	t.Run("Check: "+test.about, func(t *testing.T) {
		tt := &testingT{}
		ok := qt.Check(tt, test.got, test.checker, test.comment...)
		checkResult(t, ok, tt.errorString(), test.expectedFailure)
		if tt.fatalString() != "" {
			t.Fatalf("no fatal messages expected, but got %q", tt.fatalString())
		}
	})

}

func checkResult(t *testing.T, ok bool, got, want string) {
	t.Helper()
	if want != "" {
		assertPrefix(t, got, want+"stack:\n")
		assertBool(t, ok, false)
		return
	}
	if got != "" {
		t.Fatalf("output:\ngot  %q\nwant empty", got)
	}
	assertBool(t, ok, true)
}

// testingT can be passed to qt.New for testing purposes.
type testingT struct {
	testing.TB

	errorBuf bytes.Buffer
	fatalBuf bytes.Buffer

	subTestResult bool
	subTestName   string
	subTestT      *testing.T

	helperCalls int
	parallel    bool
}

// Error overrides testing.TB.Error so that messages are collected.
func (t *testingT) Error(a ...any) {
	fmt.Fprint(&t.errorBuf, a...)
}

// Fatal overrides testing.TB.Fatal so that messages are collected and the
// goroutine is not killed.
func (t *testingT) Fatal(a ...any) {
	fmt.Fprint(&t.fatalBuf, a...)
}

// Parallel overrides testing.TB.Parallel in order to record the call.
func (t *testingT) Parallel() {
	t.parallel = true
}

// Helper overrides testing.TB.Helper in order to count calls.
func (t *testingT) Helper() {
	t.helperCalls += 1
}

// Fatal overrides *testing.T.Fatal so that messages are collected and the
// goroutine is not killed.
func (t *testingT) Run(name string, f func(t *testing.T)) bool {
	t.subTestName, t.subTestT = name, &testing.T{}
	f(t.subTestT)
	return t.subTestResult
}

// errorString returns the error message.
func (t *testingT) errorString() string {
	return t.errorBuf.String()
}

// fatalString returns the fatal error message.
func (t *testingT) fatalString() string {
	return t.fatalBuf.String()
}

// assertPrefix fails if the got value does not have the given prefix.
func assertPrefix(t testing.TB, got, prefix string) {
	t.Helper()
	if prefix == "" {
		t.Fatal("prefix: empty value provided")
	}
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf(`prefix:
got  %q
want %q
-------------------- got --------------------
%s
-------------------- want -------------------
%s
---------------------------------------------`, got, prefix, got, prefix)
	}
}

// assertBool fails if the given boolean values don't match.
func assertBool(t testing.TB, got, want bool) {
	t.Helper()
	if got != want {
		t.Fatalf("bool:\ngot  %v\nwant %v", got, want)
	}
}

// testingChecker is a quicktest.Checker used in tests. It receives the
// provided argNames, adds notes via the provided addNotes function, and when
// the check is run the provided error is returned.
type testingChecker[T any] struct {
	paramNames []string
	args       []any
	addNotes   func(note func(key string, value any))
	err        error
}

// Check implements quicktest.Checker by returning the stored error.
func (c *testingChecker[T]) Check(got T, note func(key string, value any)) error {
	if c.addNotes != nil {
		c.addNotes(note)
	}
	return c.err
}

// ParamNames implements quicktest.Checker by returning the stored param names..
func (c *testingChecker[T]) ParamNames() []string {
	return c.paramNames
}

// Info implements quicktest.Checker by returning the stored args.
func (c *testingChecker[T]) Args() []any {
	return c.args
}
