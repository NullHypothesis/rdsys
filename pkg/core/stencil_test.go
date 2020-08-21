package core

import (
	"math/rand"
	"testing"
)

func TestContains(t *testing.T) {
	i := Interval{2, 4, "foo"}

	if i.Contains(1) || i.Contains(5) {
		t.Errorf("interval falsely claims to contain given values")
	}
	if !i.Contains(2) || !i.Contains(3) || !i.Contains(4) {
		t.Errorf("interval falsely claims to not contain given values")
	}
}

func TestFindByValue(t *testing.T) {
	s := Stencil{}

	i1 := &Interval{1, 5, "foo"}
	i2 := &Interval{6, 10, "bar"}
	s.AddInterval(i1)
	s.AddInterval(i2)

	i, err := s.FindByValue(1)
	if i != i1 {
		t.Errorf("returned incorrect interval")
	}

	if _, err = s.FindByValue(0); err == nil {
		t.Errorf("failed to return error when asked to look for non-existing interval")
	}
}

func TestGetUpperEnd(t *testing.T) {
	s := Stencil{}

	if _, err := s.GetUpperEnd(); err == nil {
		t.Errorf("failed to return error for empty stencil")
	}

	s.AddInterval(&Interval{0, 4, "foo"})
	s.AddInterval(&Interval{5, 14, "bar"})

	end, _ := s.GetUpperEnd()
	if end != 14 {
		t.Errorf("returned incorrect upper end")
	}
}

func TestGetFilterFunc(t *testing.T) {
	s := Stencil{}
	// "foo" is half as likely to get resources as "bar".
	s.AddInterval(&Interval{0, 4, "foo"})
	s.AddInterval(&Interval{5, 14, "bar"})

	f, err := s.GetFilterFunc("foo")
	if err != nil {
		t.Errorf("failed to create filter function")
	}

	// Use Monte Carlo simulation to test if the proportions behave as they
	// should.
	hits := 0
	runs := 10000
	d := &Dummy{}
	for i := 0; i < runs; i++ {
		d.UniqueId = Hashkey(rand.Uint64())
		if f(d) {
			hits++
		}
	}

	// Did we get more or less the right number of hits?
	tolerance := 500
	lowerLimit := hits + hits*2 - tolerance
	upperLimit := hits + hits*2 + tolerance

	if runs <= lowerLimit {
		t.Errorf("got unexpectedly small number of hits")
	}
	if runs >= upperLimit {
		t.Errorf("got unexpectedly large number of hits")
	}
}
