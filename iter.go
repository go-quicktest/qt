// Licensed under the MIT license, see LICENSE file for details.

package qt

import (
	"fmt"
	"reflect"
)

// containerIter provides an interface for iterating over a container
// (map, slice or array).
type containerIter[T any] interface {
	// next advances to the next item in the container.
	next() bool
	// key returns the current key as a string.
	key() string
	// value returns the current value.
	value() T
}

// newIter returns an iterator over x which must be a map, slice
// or array.
func newIter[T any](x any) (containerIter[T], error) {
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Map:
		if got, want := typeOf[T](), v.Type().Elem(); got != want {
			return nil, fmt.Errorf("map value element has unexpected type; got %v want %v", got, want)
		}
		return &mapValueIter[T]{
			iter: v.MapRange(),
		}, nil
	case reflect.Slice, reflect.Array:
		if got, want := typeOf[T](), v.Type().Elem(); got != want {
			return nil, fmt.Errorf("slice or array element has unexpected type; got %v want %v", got, want)
		}
		return &sliceIter[T]{
			slice: v,
			index: -1,
		}, nil
	default:
		return nil, fmt.Errorf("map, slice or array required")
	}
}

// sliceIter implements containerIter for slices and arrays.
type sliceIter[T any] struct {
	slice reflect.Value
	index int
	v     T
}

func (i *sliceIter[T]) next() bool {
	i.index++
	return i.index < i.slice.Len()
}

func (i *sliceIter[T]) value() T {
	return valueAs[T](i.slice.Index(i.index))
}

func (i *sliceIter[T]) key() string {
	return fmt.Sprintf("index %d", i.index)
}

type mapValueIter[T any] struct {
	iter *reflect.MapIter
	v    T
}

func (i mapValueIter[T]) next() bool {
	return i.iter.Next()
}

func (i mapValueIter[T]) key() string {
	return fmt.Sprintf("key %#v", i.iter.Key())
}

func (i mapValueIter[T]) value() T {
	return valueAs[T](i.iter.Value())
}
