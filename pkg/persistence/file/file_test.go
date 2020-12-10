package file

import (
	"log"
	"os"
	"testing"
)

type Struct struct {
	Foo string
	Bar int
}

func TestNew(t *testing.T) {

	p := New("foo", "dir")
	expected := "dir/file-foo.bin"
	if p.filename != expected {
		t.Fatalf("expected %s but got %s", expected, p.filename)
	}
}

func TestSaveLoad(t *testing.T) {

	p := New("foo", ".")
	defer func() {
		os.Remove(p.filename)
	}()

	s1 := &Struct{Foo: "foo", Bar: 1234}
	s2 := &Struct{}

	p.Save(s1)
	p.Load(s2)

	if s1.Foo != s2.Foo || s1.Bar != s2.Bar {
		log.Fatal("failed to save/load struct")
	}
}
