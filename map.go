package validator

import (
	"context"
	"errors"

	"golang.org/x/text/message"
)

// Map returns the validator to verify that the map satisfies rules constrated with build.
//
// Three named args are available in its error format.
//   - key: the name of the key of the map (type K)
//   - value: the value corresponding to the key (type V)
//   - error: occurred validation error(s) (type error)
func Map[M ~map[K]V, K comparable, V any](vs ...KeyValueValidator[V]) Validator[M] {
	var r mapValidator[M, K, V]
	for _, v := range vs {
		r.addValidator(v)
	}
	return &mapValidator[M, K, V]{
		vs: vs,
	}
}

// mapValidator represents the validator to check key-value pairs in a map.
type mapValidator[M ~map[K]V, K comparable, V any] struct {
	kvs    map[K]KeyValueValidator[V] // for specific key
	avs    []KeyValueValidator[V] // for any
	format *errorFormat
}

func (r *mapValidator[M, K, V]) addValidator(v KeyValueValidator[V]) {
	if k, ok := v.key(); ok {
		if r.kvs == nil {
			r.kvs = make(map[K]KeyValueValidator[V])
		}
		if _, ok := r.kvs[k]; ok {
			panic("the key is already added")
		}
		r.kvs[k] = v
	} else {
		r.avs = append(r.avs, v)
	}
}

// WithFormat returns shallow copy of r with its error format changed to key.
func (r *mapValidator[M, K, V]) WithFormat(key message.Reference, a ...Arg) Validator[M] {
	rr := *r
	rr.format = &errorFormat{Key: key, Args: a}
	return &rr
}

// Validate validates m.
func (r *mapValidator[M, K, V]) Validate(ctx context.Context, m M) error {
	var errs map[K]error
	keys := mapKeys(m)
	for k, vs := range r.kvs {
		v := m[k]
		if err := r.validateOf(ctx, k, v, vs); err != nil {
			errs[k] = err
		}
		delete(keys, k)
	}
	for k := range keys {
		v := 
	}
	if len(errs) > 0 {
		return &MapError[M, K, V]{
			Errors: errs,
		}
	}
	return nil
}

func mapKeys[M ~map[K]V, K comparable, V any](m M) map[K]struct{} {
	s := make(map[K]struct{})
	for k := range m {
		s[k] = struct{}{}
	}
	return s
}

func (r *mapValidator[M, K, V]) validateOf(ctx context.Context, k K, v V, vs []Validator[V]) error {
	var errs []error
	for _, rule := range vs {
		if err := rule.Validate(ctx, v); err != nil {
			err = wrapErrors(err, func(err error) error {
				e := &mapKeyError[K, V]{
					Key:   k,
					Value: v,
					Err:   err,
				}
				return errors.New(ctxPrint(ctx, e, r.format.Key, r.format.Args))
			})
			errs = append(errs, err)
		}
	}
	return joinErrors(errs...)
}

// MapError reports an error is caused in Map validator.
type MapError[M ~map[K]V, K comparable, V any] struct {
	Errors map[K]error
}

// Error implements the error interface.
func (e MapError[M, K, V]) Error() string {
	return joinErrors(e.Unwrap()...).Error()
}

// Unwrap returns each errors of err.
func (e MapError[M, K, V]) Unwrap() []error {
	return mapToSlice(e.Errors)
}

var (
	_ Validator[map[string]int] = (*mapValidator[map[string]int, string, int])(nil)
	_ Error                     = (*MapError[map[string]int, string, int])(nil)
)

// MapRule is the interface to add validators for keys to validate its value.
type MapRule[K comparable, V any] interface {
	add(k K, vs []Validator[V])
}

func AddKey[K comparable, V any](m MapRule[K, V], key K, vs ...Validator[V]) {
	m.add(key, vs)
}

// mapKeyError reports an error is caused in a certin key.
type mapKeyError[K comparable, V any] struct {
	Key   K     `arg:"key"`
	Value V     `arg:"value"`
	Err   error `arg:"error"`
}

func For[K comparable, V any](key K, vs ...Validator[V]) KeyValueValidator[K, V] {
}

type mapKeyValidator[K comparable, V any] struct {
	key K
	vs  []Validator[V]
}

func (r *mapKeyValidator[K, V]) key() (K, bool) {
	return r.key, true
}

func (r *mapKeyValidator[K, V]) Validate(ctx context.Context, v V) error {
}
