package coerce

import (
	"reflect"
	"testing"
)

type foo []int

func Test_coerce(t *testing.T) {

	type x struct {
		intslice foo
		Boolval  bool
		s        string
	}

	mymap := map[string]interface{}{
		"--intslice": []string{"5", "-12", "0.5k"},
		"--Boolval":  true,
		"-s":         nil,
	}

	var myx x
	myx.s = "hello"

	expected := x{
		intslice: []int{5, -12, 512},
		Boolval:  true,
		s:        "",
	}

	err := Coerce(&myx, mymap, "--%s", "-%s")

	if err != nil {
		t.Errorf("Test_coerce: %v", err)
	}

	if !reflect.DeepEqual(myx, expected) {
		t.Errorf("Test_coerce: got %v, expected %v\n", myx, expected)
	}

}

func Test_unmarshall(t *testing.T) {

	var i int
	err := Unmarshall(&i, "3")

	if err != nil {
		t.Errorf("Test_unmarshall: %v", err)
	}

	if i != 3 {
		t.Errorf("Test_unmarshall: expected 3, got %v", i)
	}
}
