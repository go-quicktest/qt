// Licensed under the MIT license, see LICENSE file for details.

package qt

import "errors"

// TODO move this code into checker.go

// ErrorAs checks that the error is or wraps a specific error type. If so, it
// assigns it to the provided pointer. This is analogous to calling errors.As.
//
// For instance:
//
//     // Checking for a specific error type
//     c.Assert(err, qt.ErrorAs, new(*os.PathError))
//     c.Assert(err, qt.ErrorAs, (*os.PathError)(nil))
//
//     // Checking fields on a specific error type
//     var pathError *os.PathError
//     if c.Check(err, qt.ErrorAs, &pathError) {
//         c.Assert(pathError.Path, Equals, "some_path")
//     }
//
func ErrorAs[T any](got error, want *T) Checker {
	return &errorAsChecker[T]{
		got:  got,
		want: want,
	}
}

type errorAsChecker[T any] struct {
	got  error
	want *T
}

// Check implements Checker.Check by checking that got is an error whose error
// chain matches args[0] and assigning it to args[0].
func (c *errorAsChecker[T]) Check(note func(key string, value any)) (err error) {
	if c.got == nil {
		return errors.New("got nil error but want non-nil")
	}
	gotErr := c.got.(error)
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

func (c *errorAsChecker[T]) Args() []Arg {
	return []Arg{{
		Name:  "got",
		Value: c.got,
	}, {
		Name:  "as type",
		Value: Unquoted(typeOf[T]().String()),
	}}
}

// ErrorIs returns a checker that checks that the error is or wraps a specific error value. This is
// analogous to calling errors.Is.
//
// For instance:
//
//     c.Assert(err, qt.ErrorIs, os.ErrNotExist)
//
func ErrorIs(got, want error) Checker {
	return &errorIsChecker{
		argPair: argPairOf(got, want),
	}
}

type errorIsChecker struct {
	argPair[error, error]
}

// Check implements Checker.Check by checking that got is an error whose error
// chain matches args[0].
func (c *errorIsChecker) Check(note func(key string, value any)) error {
	if c.got == nil {
		return errors.New("got nil error but want non-nil")
	}
	if !errors.Is(c.got, c.want) {
		return errors.New("wanted error is not found in error chain")
	}
	return nil
}
