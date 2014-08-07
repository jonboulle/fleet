package unit

import (
	"reflect"
	"testing"

	"github.com/coreos/fleet/pkg"
)

func TestFakeUnitManagerEmpty(t *testing.T) {
	fum := NewFakeUnitManager()

	units, err := fum.Units()
	if err != nil {
		t.Errorf("Expected no error from Units(), got %v", err)
	}

	if !reflect.DeepEqual([]string{}, units) {
		t.Errorf("Expected no units, found %v", units)
	}
}

func TestFakeUnitManagerLoadUnload(t *testing.T) {
	fum := NewFakeUnitManager()

	err := fum.Load("hello.service", Unit{})
	if err != nil {
		t.Fatalf("Expected no error from Load(), got %v", err)
	}

	units, err := fum.Units()
	if err != nil {
		t.Fatalf("Expected no error from Units(), got %v", err)
	}
	eu := []string{"hello.service"}
	if !reflect.DeepEqual(eu, units) {
		t.Fatalf("Expected units %v, found %v", eu, units)
	}

	us, err := fum.GetUnitState("hello.service")
	if err != nil {
		t.Errorf("Expected no error from GetUnitState, got %v", err)
	}

	if us == nil {
		t.Fatalf("Expected non-nil UnitState")
	}

	eus := NewUnitState("loaded", "active", "running", "")
	if !reflect.DeepEqual(*us, *eus) {
		t.Fatalf("Expected UnitState %v, got %v", eus, *us)
	}

	fum.Unload("hello.service")

	units, err = fum.Units()
	if err != nil {
		t.Errorf("Expected no error from Units(), got %v", err)
	}

	if !reflect.DeepEqual([]string{}, units) {
		t.Errorf("Expected no units, found %v", units)
	}

	us, err = fum.GetUnitState("hello.service")
	if err != nil {
		t.Errorf("Expected no error from GetUnitState, got %v", err)
	}

	if us != nil {
		t.Fatalf("Expected nil UnitState")
	}
}

func TestFakeUnitManagerGetUnitStates(t *testing.T) {
	fum := NewFakeUnitManager()

	err := fum.Load("hello.service", Unit{})
	if err != nil {
		t.Fatalf("Expected no error from Load(), got %v", err)
	}

	states, err := fum.GetUnitStates(pkg.NewUnsafeSet("hello.service", "goodbye.service"))
	if err != nil {
		t.Fatalf("Failed calling GetUnitStates: %v", err)
	}

	expectStates := map[string]*UnitState{
		"hello.service": &UnitState{
			LoadState:   "loaded",
			ActiveState: "active",
			SubState:    "running",
			UnitName:    "hello.service",
		},
	}

	if !reflect.DeepEqual(expectStates, states) {
		t.Fatalf("Received unexpected collection of UnitStates: %#v\nExpected: %#v", states, expectStates)
	}
}
