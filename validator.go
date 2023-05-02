// Package requiring provides utilities for validating any types.
package requiring

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type Validator interface {
	Validate(v any) error
}

type rule struct {
	Name       string
	Validators []Validator
	Offset     uintptr // offset within struct, in bytes
	Index      []int   // index sequence for reflect.Type.FieldByIndex
}

func (r *rule) Validate(v any) error {
	var errs []error
	for _, p := range r.Validators {
		if err := p.Validate(v); err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", r.Name, err))
		}
	}
	return errors.Join(errs...)
}

type RuleSet struct {
	base  any
	rules map[string]*rule
}

func (s *RuleSet) Add(p any, name string, vs ...Validator) {
	off := s.offsetOf(p)
	f := lookupStructField(s.base, off)
	if s.rules == nil {
		s.rules = make(map[string]*rule)
	}
	s.rules[name] = &rule{
		Name:       name,
		Validators: vs,
		Offset:     f.Offset,
		Index:      f.Index,
	}
}

func (s *RuleSet) offsetOf(p any) uintptr {
	bp := reflect.ValueOf(s.base).Pointer()
	pp := reflect.ValueOf(p).Pointer()
	return pp - bp
}

func lookupStructField(p any, off uintptr) reflect.StructField {
	v := reflect.ValueOf(p)
	fields := reflect.VisibleFields(v.Elem().Type())
	for _, f := range fields {
		if f.Offset == off {
			return f
		}
	}
	panic("xxx")
}

func (s *RuleSet) Validate(v any) error {
	p := reflect.ValueOf(v)
	if p.Kind() == reflect.Pointer {
		p = p.Elem()
	}
	var errs []error
	for _, rule := range s.rules {
		f := p.FieldByIndex(rule.Index)
		if err := rule.Validate(f.Interface()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func Struct[T any](build func(s *RuleSet, v *T)) Validator {
	var (
		s RuleSet
		v T
	)
	s.base = &v
	build(&s, &v)
	return &s
}

type ViolationError[T any] interface {
	error
	SetPrinter(p Printer[T])
}

type Printer[T any] interface {
	Print(w io.Writer, e ViolationError[T])
}

type RangeMinViolationError[T any] struct {
	Min, Max int // zero or negative values means unlimited
	Value    T
}

func (e *RangeMinViolationError[T]) Error() string {
	var p rangeMinPrinter[T]
	var buf bytes.Buffer
	p.Print(&buf, *e)
	return buf.String()
}

type notEmptyPrinter[T any] struct{}

func (notEmptyPrinter[T]) Print(w io.Writer, e RangeMinViolationError[T]) {
	fmt.Fprintf(w, "requires")
}

type rangeMinPrinter[T any] struct{}

func (rangeMinPrinter[T]) Print(w io.Writer, e RangeMinViolationError[T]) {
	fmt.Fprintf(w, "requires")
}

type notEmptyValidator[T ~string] struct {
	p Printer[T]
}

func (f *notEmptyValidator[T]) SetPrinter(p Printer[T]) {
	f.p = p
}

func (f *notEmptyValidator[T]) Validate(v any) error {
	s := v.(T)
	if s == "" {
		return &RangeMinViolationError[T]{
			Min:   1,
			Value: s,
		}
	}
	return nil
}

var (
	NotEmpty Validator = &notEmptyValidator[string]{}
)
