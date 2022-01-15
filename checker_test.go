// Licensed under the MIT license, see LICENSE file for details.

package qt_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/go-quicktest/qt"
)

type errTarget struct {
	msg string
}

func (e *errTarget) Error() string {
	return e.msg
}

var (
	targetErr = &errTarget{msg: "target"}
)

// Fooer is an interface for testing.
type Fooer interface {
	Foo()
}

type cmpType struct {
	Strings []any
	Ints    []int
}

var (
	goTime = time.Date(2012, 3, 28, 0, 0, 0, 0, time.UTC)
	chInt  = func() chan int {
		ch := make(chan int, 4)
		ch <- 42
		ch <- 47
		return ch
	}()
	sameInts = cmpopts.SortSlices(func(x, y int) bool {
		return x < y
	})
	cmpEqualsGot = cmpType{
		Strings: []any{"who", "dalek"},
		Ints:    []int{42, 47},
	}
	cmpEqualsWant = cmpType{
		Strings: []any{"who", "dalek"},
		Ints:    []int{42},
	}
)

type InnerJSON struct {
	First  string
	Second int             `json:",omitempty" yaml:",omitempty"`
	Third  map[string]bool `json:",omitempty" yaml:",omitempty"`
}

type OuterJSON struct {
	First  float64
	Second []*InnerJSON `json:"Last,omitempty" yaml:"last,omitempty"`
}

type boolean bool

type checkerTest[T any] struct {
	about                 string
	checker               qt.Checker[T]
	got                   T
	verbose               bool
	expectedCheckFailure  string
	expectedNegateFailure string
}

var checkerTests = []testRunner{checkerTest[int]{
	about:   "Equals: same values",
	checker: qt.Equals(42),
	got:     42,
	expectedNegateFailure: `
error:
  unexpected success
got:
  int(42)
want:
  <same as "got">
`,
}, checkerTest[string]{
	about:   "Equals: different values",
	checker: qt.Equals("47"),
	got:     "42",
	expectedCheckFailure: `
error:
  values are not equal
got:
  "42"
want:
  "47"
`,
}, checkerTest[string]{
	about:   "Equals: different strings with quotes",
	checker: qt.Equals(`string "bar"`),
	got:     `string "foo"`,
	expectedCheckFailure: tilde2bq(`
error:
  values are not equal
got:
  ~string "foo"~
want:
  ~string "bar"~
`),
}, checkerTest[string]{
	about:   "Equals: same multiline strings",
	checker: qt.Equals("a\nmultiline\nstring"),
	got:     "a\nmultiline\nstring",
	expectedNegateFailure: `
error:
  unexpected success
got:
  "a\nmultiline\nstring"
want:
  <same as "got">
`}, checkerTest[string]{
	about:   "Equals: different multi-line strings",
	checker: qt.Equals("just\na\nlong\nmulti-line\nstring\n"),
	got:     "a\nlong\nmultiline\nstring",
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not equal
line diff (-got +want):
%s
got:
  "a\nlong\nmultiline\nstring"
want:
  "just\na\nlong\nmulti-line\nstring\n"
`, diff([]string{"a\n", "long\n", "multiline\n", "string"}, []string{"just\n", "a\n", "long\n", "multi-line\n", "string\n", ""})),
}, checkerTest[string]{
	about:   "Equals: different single-line strings ending with newline",
	checker: qt.Equals("bar\n"),
	got:     "foo\n",
	expectedCheckFailure: `
error:
  values are not equal
got:
  "foo\n"
want:
  "bar\n"
`,
}, checkerTest[string]{
	about:   "Equals: different strings starting with newline",
	checker: qt.Equals("\nbar"),
	got:     "\nfoo",
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not equal
line diff (-got +want):
%s
got:
  "\nfoo"
want:
  "\nbar"
`, diff([]string{"\n", "foo"}, []string{"\n", "bar"})),
}, checkerTest[any]{
	about:   "Equals: different types",
	checker: qt.Equals(any("42")),
	got:     42,
	expectedCheckFailure: `
error:
  values are not equal
got:
  int(42)
want:
  "42"
`}, checkerTest[any]{
	about:   "Equals: nil and nil",
	checker: qt.Equals(any(nil)),
	got:     nil,
	expectedNegateFailure: `
error:
  unexpected success
got:
  nil
want:
  <same as "got">
`,
}, checkerTest[error]{
	about:   "Equals: error is not nil",
	checker: qt.Equals(error(nil)),
	got:     errBadWolf,
	expectedCheckFailure: `
error:
  got non-nil error
got:
  bad wolf
    file:line
want:
  nil
`}, checkerTest[error]{
	about:   "Equals: error is not nil: not formatted",
	checker: qt.Equals(error(nil)),
	got: &errTest{
		msg: "bad wolf",
	},
	expectedCheckFailure: `
error:
  got non-nil error
got:
  e"bad wolf"
want:
  nil
`,
}, checkerTest[error]{
	about:   "Equals: error does not guard against nil",
	checker: qt.Equals(error(nil)),
	got:     (*errTest)(nil),
	expectedCheckFailure: `
error:
  got non-nil error
got:
  e<nil>
want:
  nil
`,
}, checkerTest[error]{
	about:   "Equals: error is not nil: not formatted and with quotes",
	checker: qt.Equals(error(nil)),
	got: &errTest{
		msg: `failure: "bad wolf"`,
	},
	expectedCheckFailure: tilde2bq(`
error:
  got non-nil error
got:
  e~failure: "bad wolf"~
want:
  nil
`),
}, checkerTest[error]{
	about:   "Equals: different errors with same message",
	checker: qt.Equals(errors.New("bad wolf")),
	got: &errTest{
		msg: "bad wolf",
	},
	expectedCheckFailure: `
error:
  values are not equal
got type:
  *qt_test.errTest
want type:
  *errors.errorString
got:
  e"bad wolf"
want:
  <same as "got">
`,
}, checkerTest[any]{
	about:   "Equals: nil struct",
	checker: qt.Equals[any](nil),
	got:     (*struct{})(nil),
	expectedCheckFailure: `
error:
  values are not equal
got:
  (*struct {})(nil)
want:
  nil
`,
}, checkerTest[bool]{
	about:   "Equals: different booleans",
	checker: qt.Equals(false),
	got:     true,
	expectedCheckFailure: `
error:
  values are not equal
got:
  bool(true)
want:
  bool(false)
`,
}, checkerTest[any]{
	about: "Equals: uncomparable types",
	checker: qt.Equals[any](struct {
		Ints []int
	}{
		Ints: []int{42, 47},
	}),
	got: struct {
		Ints []int
	}{
		Ints: []int{42, 47},
	},
	expectedCheckFailure: `
error:
  runtime error: comparing uncomparable type struct { Ints []int }
got:
  struct { Ints []int }{
      Ints: {42, 47},
  }
want:
  <same as "got">
`}, checkerTest[cmpType]{
	about:   "DeepEquals: same values",
	checker: qt.DeepEquals(cmpEqualsGot),
	got:     cmpEqualsGot,
	expectedNegateFailure: `
error:
  unexpected success
got:
  qt_test.cmpType{
      Strings: {
          "who",
          "dalek",
      },
      Ints: {42, 47},
  }
want:
  <same as "got">
`,
}, checkerTest[cmpType]{
	about:   "DeepEquals: different values",
	checker: qt.DeepEquals(cmpEqualsWant),
	got:     cmpEqualsGot,
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not deep equal
diff (-got +want):
%s
`, diff(cmpEqualsGot, cmpEqualsWant)),
}, checkerTest[cmpType]{
	about:   "DeepEquals: different values: verbose",
	checker: qt.DeepEquals(cmpEqualsWant),
	got:     cmpEqualsGot,
	verbose: true,
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not deep equal
diff (-got +want):
%s
got:
  qt_test.cmpType{
      Strings: {
          "who",
          "dalek",
      },
      Ints: {42, 47},
  }
want:
  qt_test.cmpType{
      Strings: {
          "who",
          "dalek",
      },
      Ints: {42},
  }
`, diff(cmpEqualsGot, cmpEqualsWant)),
}, checkerTest[[]int]{
	about:   "DeepEquals: same values with options",
	checker: qt.DeepEquals([]int{3, 2, 1}, sameInts),
	got:     []int{1, 2, 3},
	expectedNegateFailure: `
error:
  unexpected success
got:
  []int{1, 2, 3}
want:
  []int{3, 2, 1}
`,
}, checkerTest[[]int]{
	about:   "DeepEquals: different values with options",
	checker: qt.DeepEquals([]int{3, 2, 1}, sameInts),
	got:     []int{1, 2, 4},
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not deep equal
diff (-got +want):
%s
`, diff([]int{1, 2, 4}, []int{3, 2, 1}, sameInts)),
}, checkerTest[[]int]{
	about:   "DiffEquals: different values with options: verbose",
	checker: qt.DeepEquals([]int{3, 2, 1}, sameInts),
	got:     []int{1, 2, 4},
	verbose: true,
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not deep equal
diff (-got +want):
%s
got:
  []int{1, 2, 4}
want:
  []int{3, 2, 1}
`, diff([]int{1, 2, 4}, []int{3, 2, 1}, sameInts)),
}, checkerTest[struct{ answer int }]{
	about: "DeepEquals: structs with unexported fields not allowed",
	checker: qt.DeepEquals(
		struct{ answer int }{
			answer: 42,
		},
	),
	got: struct{ answer int }{
		answer: 42,
	},
	expectedCheckFailure: `
error:
  cannot handle unexported field at root.answer:
  	"github.com/go-quicktest/qt_test".(struct { answer int })
  consider using a custom Comparer; if you control the implementation of type, you can also consider using an Exporter, AllowUnexported, or cmpopts.IgnoreUnexported
got:
  struct { answer int }{answer:42}
want:
  <same as "got">
`,
}, checkerTest[struct{ answer int }]{
	about: "DeepEquals: structs with unexported fields ignored",
	checker: qt.DeepEquals(
		struct{ answer int }{
			answer: 42,
		}, cmpopts.IgnoreUnexported(struct{ answer int }{})),
	got: struct{ answer int }{
		answer: 42,
	},
	expectedNegateFailure: `
error:
  unexpected success
got:
  struct { answer int }{answer:42}
want:
  <same as "got">
`,
}, checkerTest[time.Time]{
	about:   "DeepEquals: same times",
	checker: qt.DeepEquals(goTime),
	got:     goTime,
	expectedNegateFailure: `
error:
  unexpected success
got:
  s"2012-03-28 00:00:00 +0000 UTC"
want:
  <same as "got">
`,
}, checkerTest[time.Time]{
	about:   "DeepEquals: different times: verbose",
	checker: qt.DeepEquals(goTime),
	got:     goTime.Add(24 * time.Hour),
	verbose: true,
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not deep equal
diff (-got +want):
%s
got:
  s"2012-03-29 00:00:00 +0000 UTC"
want:
  s"2012-03-28 00:00:00 +0000 UTC"
`, diff(goTime.Add(24*time.Hour), goTime)),
}, checkerTest[[]string]{
	about:   "ContentEquals: same values",
	checker: qt.ContentEquals([]string{"these", "are", "the", "voyages"}),
	got:     []string{"these", "are", "the", "voyages"},
	expectedNegateFailure: `
error:
  unexpected success
got:
  []string{"these", "are", "the", "voyages"}
want:
  <same as "got">
`,
}, checkerTest[[]int]{
	about:   "ContentEquals: same contents",
	checker: qt.ContentEquals([]int{3, 2, 1}),
	got:     []int{1, 2, 3},
	expectedNegateFailure: `
error:
  unexpected success
got:
  []int{1, 2, 3}
want:
  []int{3, 2, 1}
`,
}, checkerTest[[]struct {
	Strings []any
	Ints    []int
}]{
	about: "ContentEquals: same contents on complex slice",
	checker: qt.ContentEquals(
		[]struct {
			Strings []any
			Ints    []int
		}{cmpEqualsWant, cmpEqualsGot, cmpEqualsGot},
	),
	got: []struct {
		Strings []any
		Ints    []int
	}{cmpEqualsGot, cmpEqualsGot, cmpEqualsWant},
	expectedNegateFailure: `
error:
  unexpected success
got:
  []struct { Strings []interface {}; Ints []int }{
      {
          Strings: {
              "who",
              "dalek",
          },
          Ints: {42, 47},
      },
      {
          Strings: {
              "who",
              "dalek",
          },
          Ints: {42, 47},
      },
      {
          Strings: {
              "who",
              "dalek",
          },
          Ints: {42},
      },
  }
want:
  []struct { Strings []interface {}; Ints []int }{
      {
          Strings: {
              "who",
              "dalek",
          },
          Ints: {42},
      },
      {
          Strings: {
              "who",
              "dalek",
          },
          Ints: {42, 47},
      },
      {
          Strings: {
              "who",
              "dalek",
          },
          Ints: {42, 47},
      },
  }
`}, checkerTest[struct {
	Nums []int
}]{
	about: "ContentEquals: same contents on a nested slice",
	checker: qt.ContentEquals(
		struct {
			Nums []int
		}{
			Nums: []int{4, 3, 2, 1},
		},
	),
	got: struct {
		Nums []int
	}{
		Nums: []int{1, 2, 3, 4},
	},
	expectedNegateFailure: `
error:
  unexpected success
got:
  struct { Nums []int }{
      Nums: {1, 2, 3, 4},
  }
want:
  struct { Nums []int }{
      Nums: {4, 3, 2, 1},
  }
`,
}, checkerTest[any]{
	about:   "ContentEquals: slices of different type",
	checker: qt.ContentEquals[any]([]any{"bad", "wolf"}),
	got:     []string{"bad", "wolf"},
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not deep equal
diff (-got +want):
%s
`, diff([]string{"bad", "wolf"}, []any{"bad", "wolf"})),
}, checkerTest[string]{
	about:   "Matches: perfect match",
	checker: qt.Matches("exterminate"),
	got:     "exterminate",
	expectedNegateFailure: `
error:
  unexpected success
got value:
  "exterminate"
regexp:
  <same as "got value">
`,
}, checkerTest[string]{
	about:   "Matches: match",
	checker: qt.Matches("these are the .*"),
	got:     "these are the voyages",
	expectedNegateFailure: `
error:
  unexpected success
got value:
  "these are the voyages"
regexp:
  "these are the .*"
`,
}, checkerTest[string]{
	about:   "Matches: mismatch",
	checker: qt.Matches("these are the voyages"),
	got:     "voyages",
	expectedCheckFailure: `
error:
  value does not match regexp
got value:
  "voyages"
regexp:
  "these are the voyages"
`,
}, checkerTest[string]{
	about:   "Matches: empty pattern",
	checker: qt.Matches(""),
	got:     "these are the voyages",
	expectedCheckFailure: `
error:
  value does not match regexp
got value:
  "these are the voyages"
regexp:
  ""
`,
}, checkerTest[string]{
	about:   "Matches: complex pattern",
	checker: qt.Matches("bad wolf|end of the .*"),
	got:     "end of the universe",
	expectedNegateFailure: `
error:
  unexpected success
got value:
  "end of the universe"
regexp:
  "bad wolf|end of the .*"
`,
}, checkerTest[string]{
	about:   "Matches: invalid pattern",
	checker: qt.Matches("("),
	got:     "voyages",
	expectedCheckFailure: `
error:
  bad check: cannot compile regexp: error parsing regexp: missing closing ): ` + "`^(()$`" + `
regexp:
  "("
`,
	expectedNegateFailure: `
error:
  bad check: cannot compile regexp: error parsing regexp: missing closing ): ` + "`^(()$`" + `
regexp:
  "("
`,
}, checkerTest[error]{
	about:   "ErrorMatches: perfect match",
	checker: qt.ErrorMatches("bad wolf"),
	got:     errBadWolf,
	expectedNegateFailure: `
error:
  unexpected success
got error:
  bad wolf
    file:line
regexp:
  "bad wolf"
`,
}, checkerTest[error]{
	about:   "ErrorMatches: match",
	checker: qt.ErrorMatches("bad .*"),
	got:     errBadWolf,
	expectedNegateFailure: `
error:
  unexpected success
got error:
  bad wolf
    file:line
regexp:
  "bad .*"
`,
}, checkerTest[error]{
	about:   "ErrorMatches: mismatch",
	checker: qt.ErrorMatches("exterminate"),
	got:     errBadWolf,
	expectedCheckFailure: `
error:
  error does not match regexp
got error:
  bad wolf
    file:line
regexp:
  "exterminate"
`,
}, checkerTest[error]{
	about:   "ErrorMatches: empty pattern",
	checker: qt.ErrorMatches(""),
	got:     errBadWolf,
	expectedCheckFailure: `
error:
  error does not match regexp
got error:
  bad wolf
    file:line
regexp:
  ""
`,
}, checkerTest[error]{
	about:   "ErrorMatches: complex pattern",
	checker: qt.ErrorMatches("bad wolf|end of the universe"),
	got:     errBadWolf,
	expectedNegateFailure: `
error:
  unexpected success
got error:
  bad wolf
    file:line
regexp:
  "bad wolf|end of the universe"
`,
}, checkerTest[error]{
	about:   "ErrorMatches: invalid pattern",
	checker: qt.ErrorMatches("("),
	got:     errBadWolf,
	expectedCheckFailure: `
error:
  bad check: cannot compile regexp: error parsing regexp: missing closing ): ` + "`^(()$`" + `
regexp:
  "("
`,
	expectedNegateFailure: `
error:
  bad check: cannot compile regexp: error parsing regexp: missing closing ): ` + "`^(()$`" + `
regexp:
  "("
`,
}, checkerTest[error]{
	about:   "ErrorMatches: nil error",
	checker: qt.ErrorMatches("some pattern"),
	got:     nil,
	expectedCheckFailure: `
error:
  got nil error but want non-nil
got error:
  nil
regexp:
  "some pattern"
`,
}, checkerTest[func()]{
	about:   "PanicMatches: perfect match",
	checker: qt.PanicMatches("error: bad wolf"),
	got:     func() { panic("error: bad wolf") },
	expectedNegateFailure: `
error:
  unexpected success
panic value:
  "error: bad wolf"
function:
  func() {...}
regexp:
  <same as "panic value">
`,
}, checkerTest[func()]{
	about:   "PanicMatches: match",
	checker: qt.PanicMatches("error: .*"),
	got:     func() { panic("error: bad wolf") },
	expectedNegateFailure: `
error:
  unexpected success
panic value:
  "error: bad wolf"
function:
  func() {...}
regexp:
  "error: .*"
`,
}, checkerTest[func()]{
	about:   "PanicMatches: mismatch",
	checker: qt.PanicMatches("error: exterminate"),
	got:     func() { panic("error: bad wolf") },
	expectedCheckFailure: `
error:
  panic value does not match regexp
panic value:
  "error: bad wolf"
function:
  func() {...}
regexp:
  "error: exterminate"
`,
}, checkerTest[func()]{
	about:   "PanicMatches: empty pattern",
	checker: qt.PanicMatches(""),
	got:     func() { panic("error: bad wolf") },
	expectedCheckFailure: `
error:
  panic value does not match regexp
panic value:
  "error: bad wolf"
function:
  func() {...}
regexp:
  ""
`,
}, checkerTest[func()]{
	about:   "PanicMatches: complex pattern",
	checker: qt.PanicMatches("bad wolf|end of the universe"),
	got:     func() { panic("bad wolf") },
	expectedNegateFailure: `
error:
  unexpected success
panic value:
  "bad wolf"
function:
  func() {...}
regexp:
  "bad wolf|end of the universe"
`,
}, checkerTest[func()]{
	about:   "PanicMatches: invalid pattern",
	checker: qt.PanicMatches("("),
	got:     func() { panic("error: bad wolf") },
	expectedCheckFailure: `
error:
  bad check: cannot compile regexp: error parsing regexp: missing closing ): ` + "`^(()$`" + `
panic value:
  "error: bad wolf"
regexp:
  "("
`,
	expectedNegateFailure: `
error:
  bad check: cannot compile regexp: error parsing regexp: missing closing ): ` + "`^(()$`" + `
panic value:
  "error: bad wolf"
regexp:
  "("
`,
}, checkerTest[func()]{
	about:   "PanicMatches: no panic",
	checker: qt.PanicMatches(".*"),
	got:     func() {},
	expectedCheckFailure: `
error:
  function did not panic
function:
  func() {...}
regexp:
  ".*"
`,
}, checkerTest[any]{
	about:   "IsNil: nil",
	checker: qt.IsNil,
	got:     nil,
	expectedNegateFailure: `
error:
  unexpected success
got:
  nil
`,
}, checkerTest[any]{
	about:   "IsNil: nil struct",
	checker: qt.IsNil,
	got:     (*struct{})(nil),
	expectedNegateFailure: `
error:
  unexpected success
got:
  (*struct {})(nil)
`,
}, checkerTest[any]{
	about:   "IsNil: nil func",
	checker: qt.IsNil,
	got:     (func())(nil),
	expectedNegateFailure: `
error:
  unexpected success
got:
  func() {...}
`,
}, checkerTest[any]{
	about:   "IsNil: nil map",
	checker: qt.IsNil,
	got:     (map[string]string)(nil),
	expectedNegateFailure: `
error:
  unexpected success
got:
  map[string]string{}
`,
}, checkerTest[any]{
	about:   "IsNil: nil slice",
	checker: qt.IsNil,
	got:     ([]int)(nil),
	expectedNegateFailure: `
error:
  unexpected success
got:
  []int(nil)
`,
}, checkerTest[any]{
	about:   "IsNil: nil error-implementing type",
	checker: qt.IsNil,
	got:     (*errTest)(nil),
	expectedCheckFailure: `
error:
  error containing nil value of type *qt_test.errTest. See https://golang.org/doc/faq#nil_error
got:
  e<nil>
`,
}, checkerTest[any]{
	about:   "IsNil: not nil",
	checker: qt.IsNil,
	got:     42,
	expectedCheckFailure: `
error:
  got non-nil value
got:
  int(42)
`,
}, checkerTest[any]{
	about:   "IsNil: error is not nil",
	checker: qt.IsNil,
	got:     errBadWolf,
	expectedCheckFailure: `
error:
  got non-nil error
got:
  bad wolf
    file:line
`,
}, checkerTest[any]{
	about:   "IsNotNil: success",
	checker: qt.IsNotNil,
	got:     42,
	expectedNegateFailure: `
error:
  got non-nil value
got:
  int(42)
`,
}, checkerTest[any]{
	about:   "IsNotNil: failure",
	checker: qt.IsNotNil,
	got:     nil,
	expectedCheckFailure: `
error:
  unexpected success
got:
  nil
`,
}, checkerTest[any]{
	about:   "HasLen: arrays with the same length",
	checker: qt.HasLen(4),
	got:     [4]string{"these", "are", "the", "voyages"},
	expectedNegateFailure: `
error:
  unexpected success
len(got):
  int(4)
got:
  [4]string{"these", "are", "the", "voyages"}
want length:
  <same as "len(got)">
`,
}, checkerTest[any]{
	about:   "HasLen: channels with the same length",
	checker: qt.HasLen(2),
	got:     chInt,
	expectedNegateFailure: fmt.Sprintf(`
error:
  unexpected success
len(got):
  int(2)
got:
  (chan int)(%v)
want length:
  <same as "len(got)">
`, chInt),
}, checkerTest[any]{
	about:   "HasLen: maps with the same length",
	checker: qt.HasLen(1),
	got:     map[string]bool{"true": true},
	expectedNegateFailure: `
error:
  unexpected success
len(got):
  int(1)
got:
  map[string]bool{"true":true}
want length:
  <same as "len(got)">
`,
}, checkerTest[any]{
	about:   "HasLen: slices with the same length",
	checker: qt.HasLen(0),
	got:     []int{},
	expectedNegateFailure: `
error:
  unexpected success
len(got):
  int(0)
got:
  []int{}
want length:
  <same as "len(got)">
`,
}, checkerTest[any]{
	about:   "HasLen: strings with the same length",
	checker: qt.HasLen(21),
	got:     "these are the voyages",
	expectedNegateFailure: `
error:
  unexpected success
len(got):
  int(21)
got:
  "these are the voyages"
want length:
  <same as "len(got)">
`,
}, checkerTest[any]{
	about:   "HasLen: arrays with different lengths",
	checker: qt.HasLen(0),
	got:     [4]string{"these", "are", "the", "voyages"},
	expectedCheckFailure: `
error:
  unexpected length
len(got):
  int(4)
got:
  [4]string{"these", "are", "the", "voyages"}
want length:
  int(0)
`,
}, checkerTest[any]{
	about:   "HasLen: channels with different lengths",
	checker: qt.HasLen(4),
	got:     chInt,
	expectedCheckFailure: fmt.Sprintf(`
error:
  unexpected length
len(got):
  int(2)
got:
  (chan int)(%v)
want length:
  int(4)
`, chInt),
}, checkerTest[any]{
	about:   "HasLen: maps with different lengths",
	checker: qt.HasLen(42),
	got:     map[string]bool{"true": true},
	expectedCheckFailure: `
error:
  unexpected length
len(got):
  int(1)
got:
  map[string]bool{"true":true}
want length:
  int(42)
`,
}, checkerTest[any]{
	about:   "HasLen: slices with different lengths",
	checker: qt.HasLen(1),
	got:     []int{42, 47},
	expectedCheckFailure: `
error:
  unexpected length
len(got):
  int(2)
got:
  []int{42, 47}
want length:
  int(1)
`,
}, checkerTest[any]{
	about:   "HasLen: strings with different lengths",
	checker: qt.HasLen(42),
	got:     "these are the voyages",
	expectedCheckFailure: `
error:
  unexpected length
len(got):
  int(21)
got:
  "these are the voyages"
want length:
  int(42)
`,
}, checkerTest[any]{
	about:   "HasLen: value without a length",
	checker: qt.HasLen(42),
	got:     42,
	expectedCheckFailure: `
error:
  bad check: first argument has no length
got:
  int(42)
`,
	expectedNegateFailure: `
error:
  bad check: first argument has no length
got:
  int(42)
`,
}, checkerTest[any]{
	about:   "Implements: implements interface",
	checker: qt.Implements[error](),
	got:     errBadWolf,
	expectedNegateFailure: `
error:
  unexpected success
got:
  bad wolf
    file:line
want interface:
  error
`,
}, checkerTest[any]{
	about:   "Implements: does not implement interface",
	checker: qt.Implements[Fooer](),
	got:     errBadWolf,
	expectedCheckFailure: `
error:
  got value does not implement wanted interface
got:
  bad wolf
    file:line
want interface:
  qt_test.Fooer
`,
}, checkerTest[any]{
	about:   "Implements: fails if got nil",
	checker: qt.Implements[Fooer](),
	got:     nil,
	expectedCheckFailure: `
error:
  got nil value but want non-nil
got:
  nil
`,
}, checkerTest[error]{
	about:   "Satisfies: success with an error",
	checker: qt.Satisfies(qt.IsBadCheck),
	got:     qt.BadCheckf("bad wolf"),
	expectedNegateFailure: `
error:
  unexpected success
arg:
  e"bad check: bad wolf"
predicate function:
  func(error) bool {...}
`,
}, checkerTest[int]{
	about:   "Satisfies: success with an int",
	checker: qt.Satisfies(func(v int) bool { return v == 42 }),
	got:     42,
	expectedNegateFailure: `
error:
  unexpected success
arg:
  int(42)
predicate function:
  func(int) bool {...}
`,
}, checkerTest[[]int]{
	about:   "Satisfies: success with nil",
	checker: qt.Satisfies(func(v []int) bool { return true }),
	got:     []int(nil),
	expectedNegateFailure: `
error:
  unexpected success
arg:
  []int(nil)
predicate function:
  func([]int) bool {...}
`,
}, checkerTest[error]{
	about:   "Satisfies: failure with an error",
	checker: qt.Satisfies(qt.IsBadCheck),
	got:     nil,
	expectedCheckFailure: `
error:
  value does not satisfy predicate function
arg:
  nil
predicate function:
  func(error) bool {...}
`,
}, checkerTest[string]{
	about:   "Satisfies: failure with a string",
	checker: qt.Satisfies(func(string) bool { return false }),
	got:     "bad wolf",
	expectedCheckFailure: `
error:
  value does not satisfy predicate function
arg:
  "bad wolf"
predicate function:
  func(string) bool {...}
`,
}, checkerTest[bool]{
	about:   "IsTrue: success",
	checker: qt.IsTrue,
	got:     true,
	expectedNegateFailure: `
error:
  unexpected success
got:
  bool(true)
want:
  <same as "got">
`,
}, checkerTest[bool]{
	about:   "IsTrue: failure",
	checker: qt.IsTrue,
	got:     false,
	expectedCheckFailure: `
error:
  values are not equal
got:
  bool(false)
want:
  bool(true)
`,
}, checkerTest[bool]{
	about:   "IsFalse: success",
	checker: qt.IsFalse,
	got:     false,
	expectedNegateFailure: `
error:
  unexpected success
got:
  bool(false)
want:
  <same as "got">
`,
}, checkerTest[bool]{
	about:   "IsFalse: failure",
	checker: qt.IsFalse,
	got:     true,
	expectedCheckFailure: `
error:
  values are not equal
got:
  bool(true)
want:
  bool(false)
`,
}, checkerTest[any]{
	about:   "StringContains match",
	checker: qt.Contains("world"),
	got:     "hello, world",
	expectedNegateFailure: `
error:
  unexpected success
container:
  "hello, world"
want:
  "world"
`,
}, checkerTest[any]{
	about:   "StringContains no match",
	checker: qt.Contains("worlds"),
	got:     "hello, world",
	expectedCheckFailure: `
error:
  no substring match found
container:
  "hello, world"
want:
  "worlds"
`}, checkerTest[any]{
	about:   "Contains match",
	checker: qt.Contains("a"),
	got:     []string{"a", "b", "c"},
	expectedNegateFailure: `
error:
  unexpected success
container:
  []string{"a", "b", "c"}
want:
  "a"
`,
}, checkerTest[any]{
	about:   "Contains with map",
	checker: qt.Contains("d"),
	got: map[string]string{
		"a": "d",
		"b": "a",
	},
	expectedNegateFailure: `
error:
  unexpected success
container:
  map[string]string{"a":"d", "b":"a"}
want:
  "d"
`,
}, checkerTest[any]{
	about:   "Contains with non-string",
	checker: qt.Contains(5),
	got:     "aa",
	expectedCheckFailure: `
error:
  bad check: strings can only contain strings, not int
`,
	expectedNegateFailure: `
error:
  bad check: strings can only contain strings, not int
`,
}, checkerTest[any]{
	about:   "All slice equals",
	checker: qt.All(qt.Equals("a")),
	got:     []string{"a", "a"},
	expectedNegateFailure: `
error:
  unexpected success
container:
  []string{"a", "a"}
want:
  "a"
`,
}, checkerTest[any]{
	about:   "All slice match",
	checker: qt.All(qt.Matches(".*e.*")),
	got:     []string{"red", "blue", "green"},
	expectedNegateFailure: `
error:
  unexpected success
container:
  []string{"red", "blue", "green"}
regexp:
  ".*e.*"
`,
	// TODO work out what to do about nested matches.
	//}, checkerTest[any]{
	//	about:   "All nested match",
	//	checker: qt.All(qt.All(qt.Matches(".*e.*"))),
	//	got:     [][]string{{"hello", "goodbye"}, {"red", "blue"}, {}},
	//	expectedNegateFailure: `
	//error:
	//  unexpected success
	//container:
	//  [][]string{
	//      {"hello", "goodbye"},
	//      {"red", "blue"},
	//      {},
	//  }
	//regexp:
	//  ".*e.*"
	//`,
	//}, checkerTest[any]{
	//	about:   "All nested mismatch",
	//	checker: qt.All(qt.All(qt.Matches(".*e.*"))),
	//	got:     [][]string{{"hello", "goodbye"}, {"black", "blue"}, {}},
	//	expectedCheckFailure: `
	//error:
	//  mismatch at index 1
	//error:
	//  mismatch at index 0
	//error:
	//  value does not match regexp
	//first mismatched element:
	//  "black"
	//`,
}, checkerTest[any]{
	about:   "All slice mismatch",
	checker: qt.All(qt.Matches(".*e.*")),
	got:     []string{"red", "black"},
	expectedCheckFailure: `
error:
  mismatch at index 1
error:
  value does not match regexp
first mismatched element:
  "black"
`,
}, checkerTest[any]{
	about:   "All slice mismatch with DeepEqual",
	checker: qt.All(qt.DeepEquals([]string{"a", "b"})),
	got:     [][]string{{"a", "b"}, {"a", "c"}},
	expectedCheckFailure: `
error:
  mismatch at index 1
error:
  values are not deep equal
diff (-got +want):
` + diff([]string{"a", "c"}, []string{"a", "b"}) + `
`,
}, checkerTest[any]{
	about:   "All with non-container",
	checker: qt.All(qt.Equals(5)),
	got:     5,
	expectedCheckFailure: `
error:
  bad check: map, slice or array required
`,
	expectedNegateFailure: `
error:
  bad check: map, slice or array required
`,
}, checkerTest[any]{
	about:   "All mismatch with map",
	checker: qt.All(qt.Matches(".*e.*")),
	got:     map[string]string{"a": "red", "b": "black"},
	expectedCheckFailure: `
error:
  mismatch at key "b"
error:
  value does not match regexp
first mismatched element:
  "black"
`}, checkerTest[any]{
	about:   "Any with non-container",
	checker: qt.Any(qt.Equals(5)),
	got:     5,
	expectedCheckFailure: `
error:
  bad check: map, slice or array required
`,
	expectedNegateFailure: `
error:
  bad check: map, slice or array required
`,
}, checkerTest[any]{
	about:   "Any no match",
	checker: qt.Any(qt.Equals(5)),
	got:     []int{},
	expectedCheckFailure: `
error:
  no matching element found
container:
  []int{}
want:
  int(5)
`,
}, checkerTest[[]byte]{
	about: "JSONEquals simple",
	checker: qt.JSONEquals(
		&OuterJSON{
			First: 47.11,
		},
	),
	got: []byte(`{"First": 47.11}`),
	expectedNegateFailure: tilde2bq(`
error:
  unexpected success
got:
  []uint8(~{"First": 47.11}~)
want:
  &qt_test.OuterJSON{
      First:  47.11,
      Second: nil,
  }
`),
}, checkerTest[[]byte]{
	about: "JSONEquals nested",
	checker: qt.JSONEquals(
		&OuterJSON{
			First: 47.11,
			Second: []*InnerJSON{
				{First: "Hello", Second: 42},
			},
		},
	),
	got: []byte(`{"First": 47.11, "Last": [{"First": "Hello", "Second": 42}]}`),
	expectedNegateFailure: tilde2bq(`
error:
  unexpected success
got:
  []uint8(~{"First": 47.11, "Last": [{"First": "Hello", "Second": 42}]}~)
want:
  &qt_test.OuterJSON{
      First:  47.11,
      Second: {
          &qt_test.InnerJSON{
              First:  "Hello",
              Second: 42,
              Third:  {},
          },
      },
  }
`),
}, checkerTest[[]byte]{
	about: "JSONEquals nested with newline",
	checker: qt.JSONEquals(
		&OuterJSON{
			First: 47.11,
			Second: []*InnerJSON{
				{First: "Hello", Second: 42},
				{First: "World", Third: map[string]bool{
					"F": false,
				}},
			},
		},
	),
	got: []byte(`{"First": 47.11, "Last": [{"First": "Hello", "Second": 42},
			{"First": "World", "Third": {"F": false}}]}`),
	expectedNegateFailure: `
error:
  unexpected success
got:
  []uint8("{\"First\": 47.11, \"Last\": [{\"First\": \"Hello\", \"Second\": 42},\n\t\t\t{\"First\": \"World\", \"Third\": {\"F\": false}}]}")
want:
  &qt_test.OuterJSON{
      First:  47.11,
      Second: {
          &qt_test.InnerJSON{
              First:  "Hello",
              Second: 42,
              Third:  {},
          },
          &qt_test.InnerJSON{
              First:  "World",
              Second: 0,
              Third:  {"F":false},
          },
      },
  }
`,
}, checkerTest[[]byte]{
	about: "JSONEquals extra field",
	checker: qt.JSONEquals(
		&OuterJSON{
			First: 2,
		},
	),
	got: []byte(`{"NotThere": 1}`),
	expectedCheckFailure: fmt.Sprintf(`
error:
  values are not deep equal
diff (-got +want):
%s
`, diff(map[string]any{"NotThere": 1.0}, map[string]any{"First": 2.0})),
}, checkerTest[[]byte]{
	about:   "JSONEquals cannot unmarshal obtained value",
	checker: qt.JSONEquals(nil),
	got:     []byte(`{"NotThere": `),
	expectedCheckFailure: fmt.Sprintf(tilde2bq(`
error:
  cannot unmarshal obtained contents: %s; "{\"NotThere\": "
got:
  []uint8(~{"NotThere": ~)
want:
  nil
`), mustJSONUnmarshalErr(`{"NotThere": `)),
}, checkerTest[[]byte]{
	about:   "JSONEquals cannot marshal expected value",
	checker: qt.JSONEquals(jsonErrorMarshaler{}),
	got:     []byte(`null`),
	expectedCheckFailure: `
error:
  bad check: cannot marshal expected contents: json: error calling MarshalJSON for type qt_test.jsonErrorMarshaler: qt json marshal error
`,
	expectedNegateFailure: `
error:
  bad check: cannot marshal expected contents: json: error calling MarshalJSON for type qt_test.jsonErrorMarshaler: qt json marshal error
`,
}, checkerTest[[]byte]{
	about:   "JSONEquals with []byte",
	checker: qt.JSONEquals(nil),
	got:     []byte("null"),
	expectedNegateFailure: `
error:
  unexpected success
got:
  []uint8("null")
want:
  nil
`,
}, checkerTest[[]byte]{
	about:   "JSONEquals with RawMessage",
	checker: qt.JSONEquals(json.RawMessage("null")),
	got:     []byte("null"),
	expectedNegateFailure: `
error:
  unexpected success
got:
  []uint8("null")
want:
  json.RawMessage("null")
`,
}, checkerTest[[]byte]{
	about: "CodecEquals with bad marshal",
	checker: qt.CodecEquals(
		nil,
		func(x any) ([]byte, error) { return []byte("bad json"), nil },
		json.Unmarshal,
	),
	got: []byte("null"),
	expectedCheckFailure: fmt.Sprintf(`
error:
  bad check: cannot unmarshal expected contents: %s
`, mustJSONUnmarshalErr("bad json")),
	expectedNegateFailure: fmt.Sprintf(`
error:
  bad check: cannot unmarshal expected contents: %s
`, mustJSONUnmarshalErr("bad json")),
}, checkerTest[[]byte]{
	about: "CodecEquals with options",
	checker: qt.CodecEquals(
		[]string{"a", "c", "z", "b"},
		json.Marshal,
		json.Unmarshal,
		cmpopts.SortSlices(func(x, y any) bool { return x.(string) < y.(string) }),
	),
	got: []byte(`["b", "z", "c", "a"]`),
	expectedNegateFailure: tilde2bq(`
error:
  unexpected success
got:
  []uint8(~["b", "z", "c", "a"]~)
want:
  []string{"a", "c", "z", "b"}
`),
}, checkerTest[error]{
	about:   "ErrorAs: exact match",
	checker: qt.ErrorAs(new(*errTarget)),
	got:     targetErr,
	expectedNegateFailure: `
error:
  unexpected success
got:
  e"target"
as:
  *qt_test.errTarget
`,
}, checkerTest[error]{
	about:   "ErrorAs: wrapped match",
	checker: qt.ErrorAs(new(*errTarget)),
	got:     fmt.Errorf("wrapped: %w", targetErr),
	expectedNegateFailure: `
error:
  unexpected success
got:
  e"wrapped: target"
as:
  *qt_test.errTarget
`,
}, checkerTest[error]{
	about:   "ErrorAs: fails if nil error",
	checker: qt.ErrorAs(new(*errTarget)),
	got:     nil,
	expectedCheckFailure: `
error:
  got nil error but want non-nil
got:
  nil
as:
  *qt_test.errTarget
`,
}, checkerTest[error]{
	about:   "ErrorAs: fails if mismatch",
	checker: qt.ErrorAs(new(*errTarget)),
	got:     errors.New("other error"),
	expectedCheckFailure: `
error:
  wanted type is not found in error chain
got:
  e"other error"
as:
  *qt_test.errTarget
`,
}, checkerTest[error]{
	about:   "ErrorAs: bad check if invalid as",
	checker: qt.ErrorAs(&struct{}{}),
	got:     targetErr,
	expectedCheckFailure: `
error:
  bad check: errors: *target must be interface or implement error
`,
	expectedNegateFailure: `
error:
  bad check: errors: *target must be interface or implement error
`,
}, checkerTest[error]{
	about:   "ErrorIs: exact match",
	checker: qt.ErrorIs(targetErr),
	got:     targetErr,
	expectedNegateFailure: `
error:
  unexpected success
got:
  e"target"
want:
  <same as "got">
`,
}, checkerTest[error]{
	about:   "ErrorIs: wrapped match",
	checker: qt.ErrorIs(targetErr),
	got:     fmt.Errorf("wrapped: %w", targetErr),
	expectedNegateFailure: `
error:
  unexpected success
got:
  e"wrapped: target"
want:
  e"target"
`,
}, checkerTest[error]{
	about:   "ErrorIs: fails if nil error",
	checker: qt.ErrorIs(targetErr),
	got:     nil,
	expectedCheckFailure: `
error:
  got nil error but want non-nil
got:
  nil
want:
  e"target"
`,
}, checkerTest[error]{
	about:   "ErrorIs: fails if mismatch",
	checker: qt.ErrorIs(targetErr),
	got:     errors.New("other error"),
	expectedCheckFailure: `
error:
  wanted error is not found in error chain
got:
  e"other error"
want:
  e"target"
`,
}}

func TestCheckers(t *testing.T) {
	for _, test := range checkerTests {
		test.run(t)
	}
}

func (test checkerTest[T]) run(t *testing.T) {
	t.Run(test.about, func(t *testing.T) {
		tt := &testingT{}
		qt.SetVerbosity(test.checker, test.verbose)
		ok := qt.Check(tt, test.got, test.checker)
		checkResult(t, ok, tt.errorString(), test.expectedCheckFailure)
	})
	t.Run("Not "+test.about, func(t *testing.T) {
		tt := &testingT{}
		qt.SetVerbosity(test.checker, test.verbose)
		ok := qt.Check(tt, test.got, qt.Not(test.checker))
		checkResult(t, ok, tt.errorString(), test.expectedNegateFailure)
	})
}

func diff(x, y any, opts ...cmp.Option) string {
	d := cmp.Diff(x, y, opts...)
	return strings.TrimSuffix(qt.Prefixf("  ", "%s", d), "\n")
}

type jsonErrorMarshaler struct{}

func (jsonErrorMarshaler) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("qt json marshal error")
}

func mustJSONUnmarshalErr(s string) error {
	var v any
	err := json.Unmarshal([]byte(s), &v)
	if err == nil {
		panic("want JSON error, got nil")
	}
	return err
}

func tilde2bq(s string) string {
	return strings.Replace(s, "~", "`", -1)
}
