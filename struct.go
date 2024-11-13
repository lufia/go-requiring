package validator

import (
	"context"
	"errors"
	"reflect"

	"golang.org/x/text/message"
)

// Struct returns the validator to verify that the struct satisfies rules constrated with build.
//
// Three named args are available in its error format.
//   - name: the registered field name (type string)
//   - value: user input
//   - error: occurred validation error(s) (type error)
func Struct[P ~*T, T any](build func(s StructRule, p P)) Validator[P] {
	var (
		s structValidator[P, T]
		v T
	)
	rule := structRule[P, T]{
		base:   &v,
		fields: make(map[string]structFieldRef),
	}
	build(&rule, &v)
	s.rule = &rule
	s.format = structFieldErrorFormat
	return &s
}

// structValidator represents the validator to check the struct satisfies its rules.
type structValidator[P ~*T, T any] struct {
	rule   *structRule[P, T]
	format *errorFormat
}

// WithFormat returns shallow copy of r with its error format changed to key.
func (r *structValidator[P, T]) WithFormat(key message.Reference, a ...Arg) Validator[P] {
	rr := *r
	rr.format = &errorFormat{Key: key, Args: a}
	return &rr
}

// Validate validates v.
func (r *structValidator[P, T]) Validate(ctx context.Context, v P) error {
	errs := make(map[string]error)
	for name, rule := range r.rule.fields {
		if err := rule.validateField(ctx, v, r.format.Key, r.format.Args); err != nil {
			errs[name] = err
		}
	}
	if len(errs) > 0 {
		return &StructError[P, T]{
			Value:  v,
			Errors: errs,
		}
	}
	return nil
}

// StructError reports an error is caused in Struct validator.
type StructError[P ~*T, T any] struct {
	Value  P
	Errors map[string]error
}

// Error implements the error interface.
func (e StructError[P, T]) Error() string {
	return joinErrors(e.Unwrap()...).Error()
}

func mapToSlice[M ~map[K]V, K comparable, V any](m M) []V {
	if len(m) == 0 {
		return nil
	}
	errs := make([]V, 0, len(m))
	for _, err := range m {
		errs = append(errs, err)
	}
	return errs
}

// Unwrap returns each errors of err.
func (e StructError[P, T]) Unwrap() []error {
	return mapToSlice(e.Errors)
}

var (
	_ Validator[*int] = (*structValidator[*int, int])(nil)
	_ Error           = (*StructError[*int, int])(nil)
)

// structRule manages its fields.
type structRule[P ~*T, T any] struct {
	base   P
	fields map[string]structFieldRef
}

// Add adds the rule.
func (r *structRule[P, T]) add(field structFieldRef) {
	offset := field.offsetFrom(r.base)
	f := lookupStructField(r.base, offset)
	field.setIndex(f.Index)
	r.fields[field.Name()] = field
}

func lookupStructField(p any, offset uintptr) reflect.StructField {
	v := reflect.ValueOf(p)
	fields := reflect.VisibleFields(v.Elem().Type())
	for _, f := range fields {
		if f.Offset == offset {
			return f
		}
	}
	panic("the pointer refers out of the struct")
}

// StructRule is the interface to add its fields.
type StructRule interface {
	add(field structFieldRef)
}

// AddField adds the p's field of the struct T.
func AddField[T any](s StructRule, p *T, name string, vs ...Validator[T]) {
	s.add(&structField[T]{
		name: name,
		p:    p,
		vs:   vs,
	})
}

type structField[T any] struct {
	name  string
	p     *T
	vs    []Validator[T]
	index []int
}

// Name returns field's name.
func (r *structField[T]) Name() string {
	return r.name
}

func (r *structField[T]) setIndex(index []int) {
	if r.index != nil {
		panic("the field is already added")
	}
	r.index = index
}

func (r *structField[T]) validateField(ctx context.Context, base any, key message.Reference, args []Arg) error {
	p := r.valueOf(base, r.index)
	v := p.(T)
	var errs []error
	for _, rule := range r.vs {
		if err := rule.Validate(ctx, v); err != nil {
			err = wrapErrors(err, func(err error) error {
				e := &structFieldError[T]{
					Name:  r.name,
					Value: v,
					Err:   err,
				}
				return errors.New(ctxPrint(ctx, e, key, args))
			})
			errs = append(errs, err)
		}
	}
	return joinErrors(errs...)
}

func wrapErrors(err error, fn func(err error) error) error {
	errs := flattenErrors(err)
	for i, err := range errs {
		errs[i] = fn(err)
	}
	return joinErrors(errs...)
}

func flattenErrors(err error) []error {
	var errs []error
	e, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return []error{err}
	}
	for _, err := range e.Unwrap() {
		errs = append(errs, flattenErrors(err)...)
	}
	return errs
}

func (f *structField[T]) offsetFrom(base any) uintptr {
	bp := reflect.ValueOf(base).Pointer()
	pp := reflect.ValueOf(f.p).Pointer()
	return pp - bp
}

func (f *structField[T]) valueOf(base any, index []int) any {
	bp := reflect.ValueOf(base)
	if bp.IsNil() {
		var t T
		return t
	}
	p := bp.Elem().FieldByIndex(index)
	return p.Interface()
}

// structFieldError reports an error is caused in Field validator.
type structFieldError[T any] struct {
	Name  string `arg:"name"`
	Value T      `arg:"value"`
	Err   error  `arg:"error"`
}

// structField is the interface that is used by StructRule.
type structFieldRef interface {
	Name() string
	setIndex(index []int)
	validateField(ctx context.Context, base any, key message.Reference, a []Arg) error
	offsetFrom(base any) uintptr
}

var _ structFieldRef = (*structField[string])(nil)
