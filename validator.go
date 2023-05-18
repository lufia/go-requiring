/*
Package validator enables strongly-typed Validate methods to validate any types.

# Builtin Validators

There are builtin validators.
  - In
  - InRange
  - Length
  - Max
  - MaxLength
  - Min
  - MinLength
  - Pattern
  - Required

Also there are few composition validators.
  - Join
  - Slice
  - Struct

# Error message

The builtin- and compositon-validators has default error messages.
Additionally these validators provides to modify its each default message to appropriate message on the situation. For example:

	v := validator.Min(3).WithFormat("%[1]v is not valid", validator.ByName("value"))

It is different for each the validator to be available argument names with ByName.
See each the validator documentation.

# Internationalization

The validators error messages are available in multiple languages.

The validator package assumes English is default language.
To switch default language to another one,
it is set Printer provided by [golang.org/x/text/message] to ctx that
will be passed to the first argument of Validate[T] method.
*/
package validator

import (
	"context"

	"golang.org/x/text/message"
)

// Validator is the interface that wraps the basic Validate method.
type Validator[T any] interface {
	Validate(ctx context.Context, v T) error
	WithFormat(key message.Reference, a ...Arg) Validator[T]
}

// Error is the interface that wraps Error method.
type Error interface {
	error
}

// Join bundles vs to a validator.
func Join[T any](vs ...Validator[T]) Validator[T] {
	var a []Validator[T]
	for _, v := range vs {
		if p, ok := v.(*joinValidator[T]); ok {
			a = append(a, p.vs...)
		} else {
			a = append(a, v)
		}
	}
	return &joinValidator[T]{vs: a}
}

type joinValidator[T any] struct {
	vs []Validator[T]
}

// WithFormat returns shallow copy of r with its error format changed to key.
//
// TODO(lufia): currently key is always ignored.
func (r *joinValidator[T]) WithFormat(key message.Reference, a ...Arg) Validator[T] {
	rr := *r
	return &rr
}

// Validate returns the all errors that v is validated with its each validator.
func (r *joinValidator[T]) Validate(ctx context.Context, v T) error {
	var errs []error
	for _, p := range r.vs {
		if err := p.Validate(ctx, v); err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrors(errs...)
}

var _ Validator[string] = (*joinValidator[string])(nil)

// OrderedMap is a map that guarantee that the iteration order of entries
// will be the order in which they were set.
type OrderedMap[K comparable, V any] struct {
	keys   []K
	values map[K]V
}

func (m *OrderedMap[K, V]) set(key K, v V) {
	if m.values == nil {
		m.values = make(map[K]V)
	}
	if _, ok := m.values[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.values[key] = v
}

// Len returns the length of m.
func (m *OrderedMap[K, V]) Len() int {
	return len(m.keys)
}

// Keys returns keys of m. These keys preserves the order in which they were set.
func (m *OrderedMap[K, V]) Keys() []K {
	return m.keys
}

// Get returns a value associated to key.
func (m *OrderedMap[K, V]) Get(key K) (V, bool) {
	v, ok := m.values[key]
	return v, ok
}
