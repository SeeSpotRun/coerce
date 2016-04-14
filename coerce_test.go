package coerce

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type foo []int

func report(err error, expected interface{}, got interface{}, t *testing.T) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	caller := runtime.FuncForPC(pc[0]).Name()

	if err != nil {
		t.Errorf("%s: %v", caller, err)
	} else if !reflect.DeepEqual(expected, got) {
		t.Errorf("%s: expected %v, got %v", caller, expected, got)
	}
}

func Test_Struct(t *testing.T) {
	return

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

	err := Struct(&myx, mymap, "--%s", "-%s")

	report(err, expected, myx, t)
}

func Test_Var_int_string(t *testing.T) {

	var i int
	err := Var(&i, "3")

	report(err, 3, i, t)
}

func Test_Var_string_int(t *testing.T) {

	s := ""
	err := Var(&s, 3)

	report(err, "3", s, t)
}

func Test_Var_string_date_string(t *testing.T) {

	s := ""

	tm := time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC)
	expected := fmt.Sprintf("%v", tm)
	err := Var(&s, tm)
	report(err, expected, s, t)

	var t2 time.Time
	err = Var(&t2, s)
	if err != nil {
		// FIXME
	} else {
		report(nil, tm, t2, t)
	}
}

func Test_string_duration_string(t *testing.T) {

	s := ""

	d1 := time.Duration(time.Nanosecond * 12345)
	err := Var(&s, d1)
	report(err, "12.345µs", s, t)

	var d2 time.Duration

	err = Var(&d2, s)
	report(err, d1, d2, t)

}

func Test_int(t *testing.T) {
	i, err := Int(float32(1))
	report(err, int(1), i, t)
}

func Test_float(t *testing.T) {
	f, err := Float32("1.23E4")
	report(err, float32(12300), f, t)

	f = 1.0 / 3.0
	s := String(f)

	f2, err := Float32(s)
	report(err, f, f2, t)
}
