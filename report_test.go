// Licensed under the MIT license, see LICENSE file for details.

package qt_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
)

// The tests in this file rely on their own source code lines.

func TestReportOutput(t *testing.T) {
	tt := &testingT{}
	qt.Assert(tt, 42, qt.Equals(47))
	want := `
error:
  values are not equal
got:
  int(42)
want:
  int(47)
stack:
  $file:17
    qt.Assert(tt, 42, qt.Equals(47))
`
	assertReport(t, tt, want)
}

func f1(t testing.TB) {
	f2(t)
}

func f2(t testing.TB) {
	qt.Assert(t, 42, qt.IsNil) // Real assertion here!
}

func TestIndirectReportOutput(t *testing.T) {
	tt := &testingT{}
	f1(tt)
	want := `
error:
  got non-nil value
got:
  int(42)
stack:
  $file:37
    qt.Assert(t, 42, qt.IsNil)
  $file:33
    f2(t)
  $file:42
    f1(tt)
`
	assertReport(t, tt, want)
}

func TestMultilineReportOutput(t *testing.T) {
	tt := &testingT{}
	qt.Assert(tt,
		"this string", // Comment 1.
		qt.Equals("another string"),
		qt.Commentf("a comment"), // Comment 2.
	) // Comment 3.
	want := `
error:
  values are not equal
comment:
  a comment
got:
  "this string"
want:
  "another string"
stack:
  $file:61
    qt.Assert(tt,
        "this string", // Comment 1.
        qt.Equals("another string"),
        qt.Commentf("a comment"), // Comment 2.
    )
`
	assertReport(t, tt, want)
}

func TestCmpReportOutput(t *testing.T) {
	tt := &testingT{}
	gotExamples := []*reportExample{{
		AnInt: 42,
	}, {
		AnInt: 47,
	}, {
		AnInt: 1,
	}, {
		AnInt: 2,
	}}
	wantExamples := []*reportExample{{
		AnInt: 42,
	}, {
		AnInt: 47,
	}, {
		AnInt: 2,
	}, {
		AnInt: 1,
	}, {}}
	checker := qt.DeepEquals(wantExamples)
	qt.SetVerbosity(checker, false)
	qt.Assert(tt, gotExamples, checker)
	want := `
error:
  values are not deep equal
diff (-got +want):
    []*qt_test.reportExample{
            &{AnInt: 42},
            &{AnInt: 47},
  +         &{AnInt: 2},
            &{AnInt: 1},
  -         &{AnInt: 2},
  +         &{},
    }
stack:
  $file:108
    qt.Assert(tt, gotExamples, checker)
`
	assertReport(t, tt, want)
}

func TestTopLevelAssertReportOutput(t *testing.T) {
	tt := &testingT{}
	qt.Assert(tt, 42, qt.Equals(47))
	want := `
error:
  values are not equal
got:
  int(42)
want:
  int(47)
stack:
  $file:130
    qt.Assert(tt, 42, qt.Equals(47))
`
	assertReport(t, tt, want)
}

func assertReport(t *testing.T, tt *testingT, want string) {
	t.Helper()
	got := strings.Replace(tt.fatalString(), "\t", "        ", -1)
	// go-cmp can include non-breaking spaces in its output.
	got = strings.Replace(got, "\u00a0", " ", -1)
	// Adjust for file names in different systems.
	_, file, _, ok := runtime.Caller(0)
	assertBool(t, ok, true)
	want = strings.Replace(want, "$file", file, -1)
	if got != want {
		t.Fatalf(`failure:
%q
%q
------------------------------ got ------------------------------
%s------------------------------ want -----------------------------
%s-----------------------------------------------------------------`,
			got, want, got, want)
	}
}

type reportExample struct {
	AnInt int
}
