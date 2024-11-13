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

When these builtin validators detects the value is invalid,
they returns just an error corresponding to the validator.
In other words, they don't return multiple errors wrapped by errors.Join.

Also there are few composition validators.
  - Join
  - Slice
  - Struct

These validators wraps multiple validators (including composition validators itself),
so it could be contained multiple errors to a returned error from them.

To get the details:

	// Join validator
	err := validator.Join(validator.Min(3)).Validate(context.Background(), 2)
	if e, ok := err.(interface{ Unwrap() []error }); ok {
		fmt.Println(e.Unwrap())
	}

	// Slice validator
	v := validator.Slice(validator.Min(3))
	err := v.Validate(context.Background(), []int{3, 2, 1})
	if e, ok := err.(*validator.SliceError[[]int, int]); ok {
		fmt.Println(e.Errors)
	}

	// Struct validator
	v := validator.Struct(func(s validator.StructRule, r *Data) {
		// ...
	})
	err := v.Validate(context.Background(), &Data{})
	if e, ok := err.(*validator.StructError[*Data, Data]); ok {
		fmt.Println(e.Errors)
	}

# Custom validator

The New utility function makes it easy to implement custom validators.

We highly recommend to set custom error message with WithFormat to that validator.
It also has default error message but it might be a unsufficient to your users.

# Error message

The builtin- and compositon-validators has default error messages.
Additionally these validators provides to modify its each default message to appropriate message on the situation.

For example:

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
	"errors"

	"golang.org/x/text/message"
)

// Validator is the interface that wraps the basic Validate method.
type Validator[T any] interface {
	Validate(ctx context.Context, v T) error
	WithFormat(key message.Reference, a ...Arg) Validator[T]
}

// KeyValueValidator is the interface that validate key/value pair.
type KeyValueValidator[K comparable, V any] interface {
	key() (K, bool)
	Validator[V]
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
	// TODO(lufia): should I use rsc.io/omap?
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

// New returns the validator to verify the value with fn.
// If fn returns false, the validator results as error.
//
// A named args is available in its error format.
//   - value: user input (type T)
func New[T any](fn func(v T) bool) Validator[T] {
	return &customValidator[T]{
		fn:     fn,
		format: customErrorFormat,
	}
}

type customValidator[T any] struct {
	fn     func(v T) bool
	format *errorFormat
}

// WithFormat returns shallow copy of r with its error format changed to key.
func (r *customValidator[T]) WithFormat(key message.Reference, a ...Arg) Validator[T] {
	rr := *r
	rr.format = &errorFormat{Key: key, Args: a}
	return &rr
}

// Validate returns the all errors that v is validated with its each validator.
func (r *customValidator[T]) Validate(ctx context.Context, v T) error {
	if !r.fn(v) {
		e := &customError[T]{
			Value: v,
		}
		return errors.New(ctxPrint(ctx, e, r.format.Key, r.format.Args))
	}
	return nil
}

type customError[T any] struct {
	Value T `arg:"value"`
}

var _ Validator[string] = (*customValidator[string])(nil)
