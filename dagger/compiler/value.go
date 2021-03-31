package compiler

import (
	"sort"

	"cuelang.org/go/cue"
	cueformat "cuelang.org/go/cue/format"
)

// Value is a wrapper around cue.Value.
// Use instead of cue.Value and cue.Instance
type Value struct {
	val  cue.Value
	cc   *Compiler
	inst *cue.Instance
}

func (v *Value) CueInst() *cue.Instance {
	return v.inst
}

func (v *Value) Wrap(v2 cue.Value) *Value {
	return wrapValue(v2, v.inst, v.cc)
}

func wrapValue(v cue.Value, inst *cue.Instance, cc *Compiler) *Value {
	return &Value{
		val:  v,
		cc:   cc,
		inst: inst,
	}
}

// Fill the value in-place, unlike Merge which returns a copy.
func (v *Value) Fill(x interface{}) error {
	v.cc.lock()
	defer v.cc.unlock()

	// If calling Fill() with a Value, we want to use the underlying
	// cue.Value to fill.
	if val, ok := x.(*Value); ok {
		v.val = v.val.Fill(val.val)
	} else {
		v.val = v.val.Fill(x)
	}
	return v.Validate()
}

// LookupPath is a concurrency safe wrapper around cue.Value.LookupPath
func (v *Value) LookupPath(p cue.Path) *Value {
	v.cc.rlock()
	defer v.cc.runlock()

	return v.Wrap(v.val.LookupPath(p))
}

// Lookup is a helper function to lookup by path parts.
func (v *Value) Lookup(path ...string) *Value {
	return v.LookupPath(cueStringsToCuePath(path...))
}

// Get is a helper function to lookup by path string
func (v *Value) Get(target string) *Value {
	return v.LookupPath(cue.ParsePath(target))
}

// Proxy function to the underlying cue.Value
func (v *Value) Len() cue.Value {
	return v.val.Len()
}

// Proxy function to the underlying cue.Value
func (v *Value) Kind() cue.Kind {
	return v.val.Kind()
}

// Field represents a struct field
type Field struct {
	Label string
	Value *Value
}

// Proxy function to the underlying cue.Value
// Field ordering is guaranteed to be stable.
func (v *Value) Fields() ([]Field, error) {
	it, err := v.val.Fields()
	if err != nil {
		return nil, err
	}

	fields := []Field{}
	for it.Next() {
		fields = append(fields, Field{
			Label: it.Label(),
			Value: v.Wrap(it.Value()),
		})
	}

	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Label < fields[j].Label
	})

	return fields, nil
}

// Proxy function to the underlying cue.Value
func (v *Value) Struct() (*cue.Struct, error) {
	return v.val.Struct()
}

// Proxy function to the underlying cue.Value
func (v *Value) Exists() bool {
	return v.val.Exists()
}

// Proxy function to the underlying cue.Value
func (v *Value) String() (string, error) {
	return v.val.String()
}

// Proxy function to the underlying cue.Value
func (v *Value) Int64() (int64, error) {
	return v.val.Int64()
}

// Proxy function to the underlying cue.Value
func (v *Value) Path() cue.Path {
	return v.val.Path()
}

// Proxy function to the underlying cue.Value
func (v *Value) Decode(x interface{}) error {
	return v.val.Decode(x)
}

func (v *Value) List() ([]*Value, error) {
	l := []*Value{}
	it, err := v.val.List()
	if err != nil {
		return nil, err
	}
	for it.Next() {
		l = append(l, v.Wrap(it.Value()))
	}

	return l, nil
}

// FIXME: receive string path?
func (v *Value) Merge(x interface{}, path ...string) (*Value, error) {
	if xval, ok := x.(*Value); ok {
		x = xval.val
	}

	v.cc.lock()
	result := v.Wrap(v.val.Fill(x, path...))
	v.cc.unlock()

	return result, result.Validate()
}

func (v *Value) MergePath(x interface{}, p cue.Path) (*Value, error) {
	// FIXME: array indexes and defs are not supported,
	//  they will be silently converted to regular fields.
	//  eg.  `foo.#bar[0]` will become `foo["#bar"]["0"]`
	return v.Merge(x, cuePathToStrings(p)...)
}

func (v *Value) MergeTarget(x interface{}, target string) (*Value, error) {
	return v.MergePath(x, cue.ParsePath(target))
}

// Recursive concreteness check.
func (v *Value) IsConcreteR() error {
	return v.val.Validate(cue.Concrete(true))
}

func (v *Value) Walk(before func(*Value) bool, after func(*Value)) {
	// FIXME: lock?
	var (
		llBefore func(cue.Value) bool
		llAfter  func(cue.Value)
	)
	if before != nil {
		llBefore = func(child cue.Value) bool {
			return before(v.Wrap(child))
		}
	}
	if after != nil {
		llAfter = func(child cue.Value) {
			after(v.Wrap(child))
		}
	}
	v.val.Walk(llBefore, llAfter)
}

// Export concrete values to JSON. ignoring non-concrete values.
// Contrast with cue.Value.MarshalJSON which requires all values
// to be concrete.
func (v *Value) JSON() JSON {
	var out JSON
	v.val.Walk(
		func(v cue.Value) bool {
			b, err := v.MarshalJSON()
			if err == nil {
				newOut, err := out.Set(b, cuePathToStrings(v.Path())...)
				if err == nil {
					out = newOut
				}
				return false
			}
			return true
		},
		nil,
	)
	out, _ = out.Get(cuePathToStrings(v.Path())...)
	return out
}

// Check that a value is valid. Optionally check that it matches
// all the specified spec definitions..
func (v *Value) Validate() error {
	return v.val.Validate()
}

// Return cue source for this value
func (v *Value) Source() ([]byte, error) {
	v.cc.rlock()
	defer v.cc.runlock()

	return cueformat.Node(v.val.Eval().Syntax())
}

func (v *Value) IsEmptyStruct() bool {
	if st, err := v.Struct(); err == nil {
		if st.Len() == 0 {
			return true
		}
	}
	return false
}

func (v *Value) Cue() cue.Value {
	return v.val
}
