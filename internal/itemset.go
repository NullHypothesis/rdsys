package internal

import (
	"errors"
	"sync"
)

type Stringer interface {
	String() string
}

type Set struct {
	Set map[interface{}]struct{}
	m   sync.Mutex
}

func NewSet() *Set {
	s := &Set{}
	s.Set = make(map[interface{}]struct{})
	return s
}

func (s *Set) Remove(item interface{}) error {
	s.m.Lock()
	defer s.m.Unlock()

	// Does the given key exist in the set?
	if _, exists := s.Set[item]; !exists {
		return errors.New("key does not exist in set")
	}
	delete(s.Set, item)
	return nil
}

func (s *Set) Add(item interface{}) {
	s.m.Lock()
	defer s.m.Unlock()
	s.Set[item] = struct{}{}
}

func (s *Set) Contains(item interface{}) bool {
	s.m.Lock()
	defer s.m.Unlock()
	_, exists := s.Set[item]
	return exists
}

func (s *Set) Length() int {
	s.m.Lock()
	defer s.m.Unlock()
	return len(s.Set)
}
