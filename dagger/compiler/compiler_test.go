package compiler

import (
	"testing"
)

// Test that a non-existing field is detected correctly
func TestFieldNotExist(t *testing.T) {
	c := &Compiler{}
	root, err := c.Compile("test.cue", `foo: "bar"`)
	if err != nil {
		t.Fatal(err)
	}
	if v := root.Get("foo"); !v.Exists() {
		// value should exist
		t.Fatal(v)
	}
	if v := root.Get("bar"); v.Exists() {
		// value should NOT exist
		t.Fatal(v)
	}
}

// Test that a non-existing definition is detected correctly
func TestDefNotExist(t *testing.T) {
	c := &Compiler{}
	root, err := c.Compile("test.cue", `foo: #bla: "bar"`)
	if err != nil {
		t.Fatal(err)
	}
	if v := root.Get("foo.#bla"); !v.Exists() {
		// value should exist
		t.Fatal(v)
	}
	if v := root.Get("foo.#nope"); v.Exists() {
		// value should NOT exist
		t.Fatal(v)
	}
}

func TestJSON(t *testing.T) {
	c := &Compiler{}
	v, err := c.Compile("", `foo: hello: "world"`)
	if err != nil {
		t.Fatal(err)
	}
	b1 := v.JSON()
	if string(b1) != `{"foo":{"hello":"world"}}` {
		t.Fatal(b1)
	}
	// Reproduce a bug where Value.Get().JSON() ignores Get()
	b2 := v.Get("foo").JSON()
	if string(b2) != `{"hello":"world"}` {
		t.Fatal(b2)
	}
}
