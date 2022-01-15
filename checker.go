// Licensed under the MIT license, see LICENSE file for details.

package qt

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kr/pretty"
)

// Checker is implemented by types used as part of Check/Assert invocations.
// The type parameter will be the type of the first argument passed
// to Check or Assert.
type Checker[T any] interface {
	// Check checks that the provided argument passes the check.
	// On failure, the returned error is printed along with
	// the checker arguments (obtained by calling ParamNames and Args)
	// and key-value pairs added by calling the note function.
	//
	// If Check returns ErrSilent, neither the checker arguments nor
	// the error are printed; values with note are still printed.
	Check(got T, note func(key string, value any)) error

	// ParamNames returns the checker parameters, including
	// the name of the value being checked against and the
	// names of any values passed to the the checker itself.
	ParamNames() []string

	// Args returns the arguments passed to the checker.
	// This must have a length of one less than that of ParamNames.
	Args() []any
}

// Equals is a Checker checking equality of two comparable values.
//
// For instance:
//
//     c.Assert(answer, qt.Equals(42))
func Equals[T comparable](want T) Checker[T] {
	return &equalsChecker[T]{
		argNames: argNames{"got", "want"},
		want:     want,
	}
}

type equalsChecker[T comparable] struct {
	argNames
	want T
}

func (c *equalsChecker[T]) Args() []any {
	return []any{c.want}
}

func (c *equalsChecker[T]) Check(got T, note func(key string, value any)) (err error) {
	defer func() {
		// A panic is raised when the provided values are interfaces containing non-comparable values.
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()
	if got == c.want {
		return nil
	}
	// Customize error message for non-nil errors.
	if typeOf[T]() == typeOf[error]() {
		if any(c.want) == nil {
			return errors.New("got non-nil error")
		}
		if any(got) == nil {
			return errors.New("got nil error")
		}
		// Show error types when comparing errors with different types.
		gotType := reflect.TypeOf(got)
		wantType := reflect.TypeOf(c.want)
		if gotType != wantType {
			note("got type", Unquoted(gotType.String()))
			note("want type", Unquoted(wantType.String()))
		}
		return errors.New("values are not equal")
	}
	// Show line diff when comparing different multi-line strings.
	if vals, ok := any([2]T{got, c.want}).([2]string); ok {
		got, want := vals[0], vals[1]
		isMultiLine := func(s string) bool {
			i := strings.Index(s, "\n")
			return i != -1 && i < len(s)-1
		}
		if isMultiLine(got) || isMultiLine(want) {
			diff := cmp.Diff(strings.SplitAfter(got, "\n"), strings.SplitAfter(want, "\n"))
			note("line diff (-got +want)", Unquoted(diff))
		}
	}
	return errors.New("values are not equal")
}

// DeepEquals returns a Checker checking equality of two values
// using cmp.DeepEqual according to the provided compare options.
//
// Example call:
//
//     c.Assert(list, qt.DeepEquals([]int{42, 47}, cmpopts.SortSlices))
//
// It can be useful to define your own version that uses a custom
// set of compare options:
//
//	func deepEquals[T any](want T) Checker[T] {
//		return qt.DeepEquals(want, cmp.AllowUnexported(myStruct{}))
//	}
func DeepEquals[T any](want T, opts ...cmp.Option) Checker[T] {
	return &deepEqualsChecker[T]{
		argNames:  argNames{"got", "want"},
		want:      want,
		opts:      opts,
		verbosity: testing.Verbose,
	}
}

type deepEqualsChecker[T any] struct {
	argNames
	want T
	opts []cmp.Option
	verbosity
}

func (c *deepEqualsChecker[T]) Check(got T, note func(key string, value any)) (err error) {
	defer func() {
		// A panic is raised in some cases, for instance when trying to compare
		// structs with unexported fields and neither AllowUnexported nor
		// cmpopts.IgnoreUnexported are provided.
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()
	want := c.want
	if diff := cmp.Diff(got, want, c.opts...); diff != "" {
		// Only output values when the verbose flag is set.
		if c.verbosity() {
			note("diff (-got +want)", Unquoted(diff))
			return errors.New("values are not deep equal")
		}
		note("error", Unquoted("values are not deep equal"))
		note("diff (-got +want)", Unquoted(diff))
		return ErrSilent
	}
	return nil
}

func (c *deepEqualsChecker[T]) Args() []any {
	return []any{c.want}
}

// ContentEquals is like DeepEquals but any slices in the compared values will
// be sorted before being compared.
func ContentEquals[T any](got T) Checker[T] {
	return DeepEquals(got, cmpopts.SortSlices(func(x, y any) bool {
		// TODO frankban: implement a proper sort function.
		return pretty.Sprint(x) < pretty.Sprint(y)
	}))
}

// Matches is a Checker checking that the provided string or fmt.Stringer
// matches the provided regular expression pattern.
//
// For instance:
//
//     c.Assert("these are the voyages", qt.Matches, "these are .*")
//     c.Assert(net.ParseIP("1.2.3.4"), qt.Matches, "1.*")
//
func Matches(want string) Checker[string] {
	return &matchesChecker{
		want:     want,
		argNames: argNames{"got value", "regexp"},
	}
}

type matchesChecker struct {
	argNames
	want string
}

// Check implements Checker.Check by checking that got is a string or a
// fmt.Stringer and that it matches args[0].
func (c *matchesChecker) Check(got string, note func(key string, value any)) error {
	return match(got, c.want, "value does not match regexp", note)
}

func (c *matchesChecker) Args() []any {
	return []any{c.want}
}

// ErrorMatches is a Checker checking that the provided value is an error whose
// message matches the provided regular expression pattern.
//
// For instance:
//
//     c.Assert(err, qt.ErrorMatches, "bad wolf .*")
//
func ErrorMatches(want string) Checker[error] {
	return &errorMatchesChecker{
		want:     want,
		argNames: argNames{"got error", "regexp"},
	}
}

type errorMatchesChecker struct {
	want string
	argNames
}

// Check implements Checker.Check by checking that got is an error whose
// Error() matches args[0].
func (c *errorMatchesChecker) Check(got error, note func(key string, value any)) error {
	if got == nil {
		return errors.New("got nil error but want non-nil")
	}
	return match(got.Error(), c.want, "error does not match regexp", note)
}

func (c *errorMatchesChecker) Args() []any {
	return []any{c.want}
}

// PanicMatches returns Checker checking that the provided function panics with a
// message matching the provided regular expression pattern.
//
// For instance:
//
//     c.Assert(func() {panic("bad wolf ...")}, qt.PanicMatches, "bad wolf .*")
//
func PanicMatches(want string) Checker[func()] {
	return &panicMatchesChecker{
		want:     want,
		argNames: argNames{"function", "regexp"},
	}
}

type panicMatchesChecker struct {
	want string
	argNames
}

// Check implements Checker.Check by checking that got is a func() that panics
// with a message matching args[0].
func (c *panicMatchesChecker) Check(f func(), note func(key string, value any)) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			err = errors.New("function did not panic")
			return
		}
		msg := fmt.Sprint(r)
		note("panic value", msg)
		err = match(msg, c.want, "panic value does not match regexp", note)
	}()
	f()
	return nil
}

func (c *panicMatchesChecker) Args() []any {
	return []any{c.want}
}

// IsNil is a Checker checking that the provided value is nil.
//
// For instance:
//
//     qt.Assert(t, got, qt.IsNil)
//
// As a special case, if the value is nil but implements the
// error interface, it is still considered to be non-nil.
// This means that IsNil will fail on an error value that happens
// to have an underlying nil value, because that's
// invariably a mistake.
// See https://golang.org/doc/faq#nil_error.
var IsNil = Checker[any](isNilChecker{
	argNames: argNames{"got"},
})

type isNilChecker struct {
	argNames
}

func (c isNilChecker) Check(got any, note func(key string, value any)) error {
	if got == nil {
		return nil
	}
	value := reflect.ValueOf(got)
	_, isError := got.(error)
	if canBeNil(value.Kind()) && value.IsNil() {
		if isError {
			// It's an error with an underlying nil value.
			return fmt.Errorf("error containing nil value of type %T. See https://golang.org/doc/faq#nil_error", got)
		}
		return nil
	}
	if isError {
		return errors.New("got non-nil error")
	}
	return errors.New("got non-nil value")
}

func (isNilChecker) Args() []any {
	return nil
}

// IsNotNil is a Checker checking that the provided value is not nil.
// IsNotNil is the equivalent of qt.Not(qt.IsNil)
//
// For instance:
//
//     c.Assert(got, qt.IsNotNil)
//
var IsNotNil = Not(IsNil)

// HasLen is a Checker checking that the provided value has the given length.
// The value may be a slice, array, channel, map or string.
//
// For instance:
//
//     c.Assert([]int{42, 47}, qt.HasLen, 2)
//     c.Assert(myMap, qt.HasLen, 42)
//
func HasLen(n int) Checker[any] {
	return &hasLenChecker{
		want:     n,
		argNames: []string{"got", "want length"},
	}
}

type hasLenChecker struct {
	want int
	argNames
}

// Check implements Checker.Check by checking that len(got) == args[0].
func (c *hasLenChecker) Check(got any, note func(key string, value any)) (err error) {
	v := reflect.ValueOf(got)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
	default:
		note("got", got)
		return BadCheckf("first argument has no length")
	}
	length := v.Len()
	note("len(got)", length)
	if length != c.want {
		return fmt.Errorf("unexpected length")
	}
	return nil
}

func (c *hasLenChecker) Args() []any {
	return []any{c.want}
}

// Implements checks that the provided value implements the
// interface specified by the type parameter.
//
// For instance:
//
//     c.Assert(myReader, qt.Implements[io.ReadCloser]())
//
func Implements[I any]() Checker[any] {
	return &implementsChecker{
		want:     typeOf[I](),
		argNames: []string{"got", "want interface"},
	}
}

type implementsChecker struct {
	want reflect.Type
	argNames
}

var emptyInterface = reflect.TypeOf((*any)(nil)).Elem()

// Check implements Checker.Check by checking that got implements the
// interface pointed to by args[0].
func (c *implementsChecker) Check(got any, note func(key string, value any)) (err error) {
	if got == nil {
		note("error", Unquoted("got nil value but want non-nil"))
		note("got", got)
		return ErrSilent
	}
	if c.want.Kind() != reflect.Interface {
		note("want type", Unquoted(c.want.String()))
		return BadCheckf("want an interface type but a concrete type was provided")
	}

	gotType := reflect.TypeOf(got)
	if !gotType.Implements(c.want) {
		note("error", Unquoted("got value does not implement wanted interface"))
		note("got", got)
		note("want interface", Unquoted(c.want.String()))
		return ErrSilent
	}

	return nil
}

func (c *implementsChecker) Args() []any {
	return []any{Unquoted(c.want.String())}
}

// Satisfies returns a Checker checking that the provided value, when used as
// argument of the provided predicate function, causes the function to return
// true.
//
// For instance:
//
//     // Check that an error from os.Open satisfies os.IsNotExist.
//     c.Assert(err, qt.Satisfies(os.IsNotExist))
//
//     // Check that a floating point number is a not-a-number.
//     c.Assert(f, qt.Satisfies(math.IsNaN))
//
func Satisfies[T any](f func(T) bool) Checker[T] {
	return &satisfiesChecker[T]{
		predicate: f,
		argNames:  []string{"arg", "predicate function"},
	}
}

type satisfiesChecker[T any] struct {
	predicate func(T) bool
	argNames
}

// Check implements Checker.Check by checking that args[0](got) == true.
func (c *satisfiesChecker[T]) Check(got T, note func(key string, value any)) error {
	if c.predicate(got) {
		return nil
	}
	return fmt.Errorf("value does not satisfy predicate function")
}

func (c *satisfiesChecker[T]) Args() []any {
	return []any{c.predicate}
}

// IsTrue is a Checker checking that the provided value is true.
//
// For instance:
//
//     c.Assert(true, qt.IsTrue)
//
var IsTrue = Equals(true)

// IsFalse is a Checker checking that the provided value is false.
//
// For instance:
//
//     c.Assert(false, qt.IsFalse)
//     c.Assert(IsValid(), qt.IsFalse)
//
var IsFalse = Equals(false)

// cmpOption represents the cmp.Option type from the github.com/google/go-cmp/cmp
// package.
type cmpOption struct {
}

// Not returns a Checker negating the given Checker.
//
// For instance:
//
//     c.Assert(got, qt.Not(qt.IsNil))
//     c.Assert(answer, qt.Not(qt.Equals(42))
//
func Not[T any](c Checker[T]) Checker[T] {
	// Not(Not(c)) becomes c
	if c, ok := c.(notChecker[T]); ok {
		return c.Checker
	}
	return notChecker[T]{
		Checker: c,
	}
}

type notChecker[T any] struct {
	Checker[T]
}

func (c notChecker[T]) Check(got T, note func(key string, value any)) error {
	err := c.Checker.Check(got, note)
	if IsBadCheck(err) {
		return err
	}
	if err != nil {
		return nil
	}
	return errors.New("unexpected success")
}

// Contains is a checker that checks that a map, slice, array
// or string contains a value. It's the same as using
// Any(Equals), except that it has a special case
// for strings - if the first argument is a string,
// the second argument must also be a string
// and strings.Contains will be used.
//
// For example:
//
//     c.Assert("hello world", qt.Contains, "world")
//     c.Assert([]int{3,5,7,99}, qt.Contains, 7)
//
func Contains[T comparable](want T) Checker[any] {
	return &containsChecker[T]{
		want:     want,
		argNames: argNames{"container", "want"},
	}
	return Any(Equals(want))
}

type containsChecker[T comparable] struct {
	want T
	argNames
}

func (c *containsChecker[T]) Check(got any, note func(key string, value any)) error {
	if got, ok := got.(string); ok {
		want, ok := any(c.want).(string)
		if !ok {
			return BadCheckf("strings can only contain strings, not %T", c.want)
		}
		if strings.Contains(got, want) {
			return nil
		}
		return errors.New("no substring match found")
	}
	return Any(Equals(c.want)).Check(got, note)
}

func (c *containsChecker[T]) Args() []any {
	return []any{c.want}
}

// Any returns a Checker that uses the given checker to check elements
// of a slice or array or the values from a map. It succeeds if any element
// passes the check.
//
// For example:
//
//     c.Assert([]int{3,5,7,99}, qt.Any(qt.Equals(7)))
//     c.Assert([][]string{{"a", "b"}, {"c", "d"}}, qt.Any(qt.DeepEquals([]string{"c", "d"})))
//
// See also All and Contains.
func Any[Elem any](c Checker[Elem]) Checker[any] {
	return &anyChecker[Elem]{
		argNames:    append(argNames{"container"}, c.ParamNames()[1:]...),
		elemChecker: c,
	}
}

type anyChecker[T any] struct {
	argNames
	elemChecker Checker[T]
}

// Check implements Checker.Check by checking that one of the elements of
// got passes the c.elemChecker check.
func (c *anyChecker[T]) Check(got any, note func(key string, value any)) error {
	iter, err := newIter[T](got)
	if err != nil {
		return BadCheckf("%v", err)
	}
	for iter.next() {
		// For the time being, discard the notes added by the sub-checker,
		// because it's not clear what a good behaviour would be.
		// Should we print all the failed check for all elements? If there's only
		// one element in the container, the answer is probably yes,
		// but let's leave it for now.
		err := c.elemChecker.Check(
			iter.value(),
			func(key string, value any) {},
		)
		if err == nil {
			return nil
		}
		if IsBadCheck(err) {
			return BadCheckf("at %s: %v", iter.key(), err)
		}
	}
	return errors.New("no matching element found")
}

func (c *anyChecker[T]) Args() []any {
	return c.elemChecker.Args()
}

// All returns a Checker that uses the given checker to check elements
// of slice or array or the values of a map. It succeeds if all elements
// pass the check.
// On failure it prints the error from the first index that failed.
//
// For example:
//
//     c.Assert([]int{3, 5, 8}, qt.All(qt.Not(qt.Equals(0))))
//     c.Assert([][]string{{"a", "b"}, {"a", "b"}}, qt.All(qt.DeepEquals([]string{"c", "d"})))
//
// See also Any and Contains.
func All[Elem any](c Checker[Elem]) Checker[any] {
	return &allChecker[Elem]{
		argNames:    append([]string{"container"}, c.ParamNames()[1:]...),
		elemChecker: c,
	}
}

type allChecker[T any] struct {
	argNames
	elemChecker Checker[T]
}

// Check implement Checker.Check by checking that all the elements of got
// pass the c.elemChecker check.
func (c *allChecker[T]) Check(got any, notef func(key string, value any)) error {
	iter, err := newIter[T](got)
	if err != nil {
		return BadCheckf("%v", err)
	}
	for iter.next() {
		// Store any notes added by the checker so
		// we can add our own note at the start
		// to say which element failed.
		var notes []note
		err := c.elemChecker.Check(
			iter.value(),
			func(key string, val any) {
				notes = append(notes, note{key, val})
			},
		)
		if err == nil {
			continue
		}
		if IsBadCheck(err) {
			return BadCheckf("at %s: %v", iter.key(), err)
		}
		notef("error", Unquoted("mismatch at "+iter.key()))
		// TODO should we print the whole container value in
		// verbose mode?
		if err != ErrSilent {
			// If the error's not silent, the checker is expecting
			// the caller to print the error and the value that failed.
			notef("error", Unquoted(err.Error()))
			notef("first mismatched element", iter.value())
		}
		for _, n := range notes {
			notef(n.key, n.value)
		}
		return ErrSilent
	}
	return nil
}

func (c *allChecker[T]) Args() []any {
	return c.elemChecker.Args()
}

// JSONEquals is a checker that checks whether a byte slice
// is JSON-equivalent to a Go value. See CodecEquals for
// more information.
//
// It uses DeepEquals to do the comparison. If a more sophisticated
// comparison is required, use CodecEquals directly.
//
// For instance:
//
//     c.Assert(`{"First": 47.11}`, qt.JSONEquals, &MyStruct{First: 47.11})
//
func JSONEquals(want any) Checker[[]byte] {
	return CodecEquals(want, json.Marshal, json.Unmarshal)
}

// CodecEquals returns a checker that checks for codec value equivalence.
//
// It expects two arguments: a byte slice or a string containing some
// codec-marshaled data, and a Go value.
//
// It uses unmarshal to unmarshal the data into an any value.
// It marshals the Go value using marshal, then unmarshals the result into
// an any value.
//
// It then checks that the two any values are deep-equal to one
// another, using CmpEquals(opts) to perform the check.
//
// See JSONEquals for an example of this in use.
func CodecEquals(
	want any,
	marshal func(any) ([]byte, error),
	unmarshal func([]byte, any) error,
	opts ...cmp.Option,
) Checker[[]byte] {
	return &codecEqualChecker{
		want:      want,
		argNames:  argNames{"got", "want"},
		marshal:   marshal,
		unmarshal: unmarshal,
		opts:      opts,
	}
}

type codecEqualChecker struct {
	argNames
	want      any
	marshal   func(any) ([]byte, error)
	unmarshal func([]byte, any) error
	opts      []cmp.Option
}

func (c *codecEqualChecker) Check(got []byte, note func(key string, value any)) error {
	wantContentBytes, err := c.marshal(c.want)
	if err != nil {
		return BadCheckf("cannot marshal expected contents: %v", err)
	}
	var wantContentVal any
	if err := c.unmarshal(wantContentBytes, &wantContentVal); err != nil {
		return BadCheckf("cannot unmarshal expected contents: %v", err)
	}
	var gotContentVal any
	if err := c.unmarshal(got, &gotContentVal); err != nil {
		return fmt.Errorf("cannot unmarshal obtained contents: %v; %q", err, got)
	}
	return DeepEquals(wantContentVal, c.opts...).Check(gotContentVal, note)
}

func (c *codecEqualChecker) Args() []any {
	return []any{c.want}
}

// argNames helps implementing Checker.ParamNames.
type argNames []string

// ParamNames implements Checker.ParamNames by returning the argument names.
func (a argNames) ParamNames() []string {
	return a
}

type verbosity func() bool

func (v *verbosity) setVerbose(verbose bool) {
	*v = func() bool {
		return verbose
	}
}

// match checks that the given error message matches the given pattern.
func match(got string, regex string, msg string, note func(key string, value any)) error {
	matches, err := regexp.MatchString("^("+regex+")$", got)
	if err != nil {
		note("regexp", regex)
		return BadCheckf("cannot compile regexp: %s", err)
	}
	if matches {
		return nil
	}
	return errors.New(msg)
}

// canBeNil reports whether a value or type of the given kind can be nil.
func canBeNil(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}

func typeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func valueAs[T any](v reflect.Value) (r T) {
	reflect.ValueOf(&r).Elem().Set(v)
	return
}
