// Licensed under the MIT license, see LICENSE file for details.

package qtsuite_test

import (
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/go-quicktest/qt/qtsuite"
)

func TestRunSuite(t *testing.T) {
	var calls []call
	qtsuite.Run(t, testSuite{calls: &calls})
	qt.Assert(t, calls, qt.DeepEquals([]call{
		{"Test1", 0},
		{"Test4", 0},
	}))
}

func TestRunSuiteEmbedded(t *testing.T) {
	var calls []call
	suite := struct {
		testSuite
	}{testSuite: testSuite{calls: &calls}}
	qtsuite.Run(t, suite)
	qt.Assert(t, calls, qt.DeepEquals([]call{
		{"Test1", 0},
		{"Test4", 0},
	}))
}

func TestRunSuitePtr(t *testing.T) {
	var calls []call
	qtsuite.Run(t, &testSuite{calls: &calls})
	qt.Assert(t, calls, qt.DeepEquals([]call{
		{"Init", 0},
		{"Test1", 1},
		{"Init", 0},
		{"Test4", 1},
	}))
}

type testSuite struct {
	init  int
	calls *[]call
}

func (s testSuite) addCall(name string) {
	*s.calls = append(*s.calls, call{Name: name, Init: s.init})
}

func (s *testSuite) Init(*testing.T) {
	s.addCall("Init")
	s.init++
}

func (s testSuite) Test1(*testing.T) {
	s.addCall("Test1")
}

func (s testSuite) Test4(*testing.T) {
	s.addCall("Test4")
}

func (s testSuite) Testa(*testing.T) {
	s.addCall("Testa")
}

type call struct {
	Name string
	Init int
}

//func TestInvalidInit(t *testing.T) {
//	c := qt.New(t)
//	tt := &testingT{}
//	tc := qt.New(tt)
//	qtsuite.Run(tc, invalidTestSuite{})
//	c.Assert(tt.fatalString(), qt.Equals, "wrong signature for Init, must be Init(*testing.T)")
//}
//
//type invalidTestSuite struct{}
//
//func (invalidTestSuite) Init() {}
//}
