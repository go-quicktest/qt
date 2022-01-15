// Licensed under the MIT license, see LICENSE file for details.

package qt

import (
	"errors"
)

// TODO move this code into checker.go

// ErrorAs checks that the error is or wraps a specific error type. If so, it
// assigns it to the provided pointer. This is analogous to calling errors.As.
//
// For instance:
//
//     // Checking for a specific error type
//     c.Assert(err, qt.ErrorAs, new(*os.PathError))
//
//     // Checking fields on a specific error type
//     var pathError *os.PathError
//     if c.Check(err, qt.ErrorAs, &pathError) {
//         c.Assert(pathError.Path, Equals, "some_path")
//     }
//
func ErrorAs[T any](want *T) Checker[error] {
	return &errorAsChecker[T]{
		argNames: []string{"got", "as"},
		want:     want,
	}
}

type errorAsChecker[T any] struct {
	argNames
	want *T
}

// Check implements Checker.Check by checking that got is an error whose error
// chain matches args[0] and assigning it to args[0].
func (c *errorAsChecker[T]) Check(got error, note func(key string, value any)) (err error) {
	if got == nil {
		return errors.New("got nil error but want non-nil")
	}
	gotErr := got.(error)
	defer func() {
		// A panic is raised when the target is not a pointer to an interface
		// or error.
		if r := recover(); r != nil {
			err = BadCheckf("%s", r)
		}
	}()
	want := c.want
	if want == nil {
		want = new(T)
	}
	if !errors.As(gotErr, want) {
		return errors.New("wanted type is not found in error chain")
	}
	return nil
}

func (c *errorAsChecker[T]) Args() []any {
	return []any{Unquoted(typeOf[T]().String())}
}

// ErrorIs returns a checker that checks that the error is or wraps a specific error value. This is
// analogous to calling errors.Is.
//
// For instance:
//
//     c.Assert(err, qt.ErrorIs, os.ErrNotExist)
//
func ErrorIs(want error) Checker[error] {
	return &errorIsChecker{
		want:     want,
		argNames: []string{"got", "want"},
	}
}

type errorIsChecker struct {
	argNames
	want error
}

// Check implements Checker.Check by checking that got is an error whose error
// chain matches args[0].
func (c *errorIsChecker) Check(got error, note func(key string, value any)) error {
	if got == nil {
		return errors.New("got nil error but want non-nil")
	}
	if !errors.Is(got, c.want) {
		return errors.New("wanted error is not found in error chain")
	}
	return nil
}

func (c *errorIsChecker) Args() []any {
	return []any{c.want}
}
