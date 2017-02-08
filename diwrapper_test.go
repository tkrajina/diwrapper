package diwrapper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type InitializableStruct struct {
	initialized bool
}

func (s *InitializableStruct) Init() error {
	s.initialized = true
	return nil
}

var _ Initializer = (*InitializableStruct)(nil)

type StoppableStruct struct {
	Stopped bool
}

func (s *StoppableStruct) Clean() error {
	s.Stopped = true
	return nil
}

var _ Cleaner = (*StoppableStruct)(nil)

func TestSimple(t *testing.T) {
	s := InitializableStruct{}

	New().
		WithObject(&s).
		InitializeGraph()

	assert.True(t, s.initialized)
}

func TestInitialization(t *testing.T) {
	type Aaa1 struct{}
	type Bbb1 struct {
		Aaa1 *Aaa1 `inject:""`
	}

	b := new(Bbb1)

	New().
		WithObjects(new(Aaa1), b).
		InitializeGraph()

	assert.NotNil(t, b.Aaa1)
}

func TestStopping(t *testing.T) {

	obj := StoppableStruct{}

	di := New().
		WithObjects(&obj).
		InitializeGraph()

	// This will usually be called in defer:
	di.Stop()

	assert.True(t, obj.Stopped)
}

func TestNamed(t *testing.T) {
	type Aaa struct{}
	type Bbb struct {
		Aaa *Aaa `inject:"aaa"`
	}

	b := new(Bbb)

	New().
		WithNamedObject("aaa", new(Aaa)).
		WithObject(b).
		InitializeGraph()

	assert.NotNil(t, b.Aaa)
}

func TestInvalidNamed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	type Aaa struct{}
	type Bbb struct {
		Aaa *Aaa `inject:"unknown_aaa"`
	}

	b := new(Bbb)

	New().
		WithNamedObject("aaa", new(Aaa)).
		WithObject(b).
		InitializeGraph()

	assert.Fail(t, "Must panic")
}

func TestDoubleNamed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	type Aaa struct{}

	New().
		WithNamedObject("aaa", new(Aaa)).
		WithNamedObject("aaa", new(Aaa)).
		InitializeGraph()

	assert.Fail(t, "Must panic")
}
