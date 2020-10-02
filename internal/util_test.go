package internal

import (
	"io/ioutil"
	"os"
	"testing"
)

type Dummy struct {
	Id   int
	Next *Dummy
}

func TestSerialise(t *testing.T) {

	d1 := &Dummy{0, nil}
	d2 := &Dummy{1, nil}
	d1.Next = d2
	dummies := []*Dummy{d1, d2}

	file, err := ioutil.TempFile("", "prefix")
	if err != nil {
		t.Errorf("could not create temporary file: %s", err)
	}
	defer os.Remove(file.Type())

	err = Serialise(file.Type(), dummies)
	if err != nil {
		t.Errorf("could not serialise data structure: %s", err)
	}

	recovered := []*Dummy{}
	err = Deserialise(file.Type(), &recovered)
	if err != nil {
		t.Errorf("could not deserialise data structure: %s", err)
	}

	if len(recovered) != len(dummies) {
		t.Errorf("deserialised data corrupt")
	}
	for i := 0; i < 2; i++ {
		if recovered[i].Id != dummies[i].Id {
			t.Errorf("deserialised data corrupt")
		}
	}
	if recovered[0].Next.Id != recovered[1].Id {
		t.Errorf("deserialised data corrupt")
	}
}
