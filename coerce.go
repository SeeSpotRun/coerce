/**
*  coerce is free software: you can redistribute it and/or modify
*  it under the terms of the GNU General Public License as published by
*  the Free Software Foundation, either version 3 of the License, or
*  (at your option) any later version.
*
*  coerce is distributed in the hope that it will be useful,
*  but WITHOUT ANY WARRANTY; without even the implied warranty of
*  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*  GNU General Public License for more details.
*
** Authors:
 *
 *  - Daniel <SeeSpotRun> T.   2016-2016 (https://github.com/SeeSpotRun)
 *
** Hosted on https://github.com/SeeSpotRun/coerce
*
**/

package coerce

// package coerce coerces map[string]interface{} values into struct fields

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// Struct attempts to unmarshall the values in 'from' into the fields
// in the structure pointed to by 'target'.  The field names are used as
// map keys.  Optional format strings can be used to morph the field
// names into keys, eg "--%s" will map field "foo" to key "--foo".
// If more than one format is supplied, these will be tried in order
// until the first matching key is found.
// When coercing from string to any integer types, if the string ends
// with B|K|M|G|T (case-insensitive) then these will be interpreted
// as multipliers of 1, 1024, etc.
//
// Example:
//	type x struct{
//		intslice  []int
//		boolval   bool
//		s         string
//	}
//
//	mymap := map[string]interface{} {
//		"--intslice":  []string {"5", "12", "0.5k"},
//		"--boolval" :  true,
//		"-s"        :  "hello",
//	}
//
//	var myx x
//
//	err := coerce.Struct(&myx, mymap, "--%s", "-%s")
//	fmt.Println(err, myx) // <nil> {[5 12 512] true hello}
//
// Note: coercing unexported fields uses 'unsafe' pointers
//
func Struct(to interface{}, from map[string]interface{}, formats ...string) error {

	// parse errors are accumulated into errstr
	errstr := ""

	// get target as reflect.Value and check kind:
	pt := reflect.ValueOf(to)
	vt := reflect.Indirect(pt)
	if vt.Kind() != reflect.Struct || pt.Kind() != reflect.Ptr {
		return fmt.Errorf("expected *struct for 'to', got %v", pt.Kind())
	}

	// iterate over struct fields
	for i := 0; i < vt.NumField(); i++ {

		// get field type and pointer to value
		f := vt.Type().Field(i)
		vf := vt.Field(i)
		if !vf.CanSet() {
			// use 'unsafe' workaround for unexported fields:
			if string(f.Name[0]) == strings.ToLower(string(f.Name[0])) {
				pu := unsafe.Pointer(vf.Addr().Pointer())
				vf = reflect.Indirect(reflect.NewAt(vf.Type(), pu))
			}
			if !vf.CanSet() {
				errstr += "field " + f.Name + "not setable\n"
				continue
			}
		}

		// look for field name in map keys
		v, err := findVal(f.Name, from, formats)
		if err != nil {
			continue
		}

		if v == nil {
			// nil value in map - leave the field alone
			continue
		}

		vv := reflect.ValueOf(v)
		err = unmarshall(vf, vv)

		if err != nil {
			errstr += err.Error() + "\n"
		}

	}

	if errstr != "" {
		return fmt.Errorf("%s", errstr[:len(errstr)-1]) // strips trailling newline
	}
	return nil
}

// Var attempts to cast the content of 'from' into the variable pointed to by 'pto'
func Var(pto interface{}, from interface{}) error {

	return unmarshall(reflect.Indirect(reflect.ValueOf(pto)), reflect.ValueOf(from))
}

// unmarshallString parses string s to in vto
func unmarshallString(vto reflect.Value, tto reflect.Type, s string) error {

	// custom handlers for non-builtin types:
	switch tto.String() {

	case "time.Duration":
		d, e := time.ParseDuration(s)
		if e != nil {
			return e
		}
		vto.Set(reflect.ValueOf(d))
		return nil
	}

	// handle builtin types:
	switch vto.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		ival, err := strconv.ParseInt(s, 10, tto.Bits())

		if err != nil {
			// try again looking for B/K/M/G/T
			ival, err = getBytes(s, err)
			if err != nil {
				return err
			}
		}

		vto.SetInt(ival)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:

		uval, err := strconv.ParseUint(s, 10, tto.Bits())

		if err != nil {

			// try again looking for B/K/M/G/T
			ival, e := getBytes(s, err)
			if e != nil {
				return e
			}
			uval = uint64(ival)

		}

		vto.SetUint(uval)
		return nil

	case reflect.Float32, reflect.Float64:

		fval, err := strconv.ParseFloat(s, tto.Bits())

		if err != nil {
			return err
		}

		vto.SetFloat(fval)
		return nil
	}

	return fmt.Errorf("don't know how to unmarshall string to %v\n", tto)
}

// unmarshallFloat marshalls a float value into vto
func unmarshallFloat(vto reflect.Value, tto reflect.Type, f float64) error {

	switch tto.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vto.SetInt(int64(f))
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		vto.SetUint(uint64(f))
		return nil
	}

	return fmt.Errorf("don't know how to unmarshall float to %v\n", tto)
}

// unmarshall tries to parse vfrom value into vto
func unmarshall(vto reflect.Value, vfrom reflect.Value) error {

	// try for direct assign:
	tto := vto.Type()
	if vfrom.Type().AssignableTo(tto) {
		vto.Set(vfrom)
		return nil
	}

	// unmarshall from slices recursively:
	if vfrom.Kind() == reflect.Slice {
		if vto.Kind() == reflect.Slice {
			// ...to a slice:
			// set slice size:
			vto.Set(reflect.MakeSlice(vto.Type(), vfrom.Len(), vfrom.Len()))

			for j := 0; j < vfrom.Len(); j++ {
				// unmarshall slice elements
				err := unmarshall(vto.Index(j), vfrom.Index(j))
				if err != nil {
					return err
				}
			}
			return nil

		} else if vfrom.Len() == 1 {
			// tolerate mapping of slices with length==1 to a single field
			return unmarshall(vto, vfrom.Index(0))
		} else {
			return fmt.Errorf("can't coerce %v from multi-value slice", tto)
		}
	}

	// unmarshalling to string is easy: let fmt do the thinking:
	if tto.Kind() == reflect.String {
		vto.SetString(fmt.Sprintf("%v", vfrom.Interface()))
		return nil
	}

	// case-by-case for everything else:
	switch vfrom.Kind() {

	case reflect.String:
		return unmarshallString(vto, tto, vfrom.String())

	case reflect.Float32, reflect.Float64:
		return unmarshallFloat(vto, tto, vfrom.Float())

		// case Int, Uint etc should generally be handled by AssignableTo or fmt.Sprintf
	}

	return fmt.Errorf("Don't know how to unmarshall %v to %v\n", vfrom.Type(), tto)
}

// Int tries to return an int value based on content of 'from'
func Int(from interface{}) (i int, e error) {
	e = Var(&i, from)
	return
}

// Int64 tries to return an int64 value based on content of 'from'
func Int64(from interface{}) (i int64, e error) {
	e = Var(&i, from)
	return
}

// Uint tries to return a uint value based on content of 'from'
func Uint(from interface{}) (u uint, e error) {
	e = Var(&u, from)
	return
}

// Uint64 tries to return a uint64 value based on content of 'from'
func Uint64(from interface{}) (u int64, e error) {
	e = Var(&u, from)
	return
}

// Float32 tries to return a float32 value based on content of 'from'
func Float32(from interface{}) (f float32, e error) {
	e = Var(&f, from)
	return
}

// Float64 tries to return a float64 value based on content of 'from'
func Float64(from interface{}) (f float64, e error) {
	e = Var(&f, from)
	return
}

// String is equivalent to fmt.Sprint(from)
func String(from interface{}) (s string) {
	Var(&s, from)
	return
}

// findVal tries to find map key matching field name formatted as per formats
func findVal(baseName string, from map[string]interface{}, formats []string) (interface{}, error) {

	if len(formats) == 0 {
		// handle case where no formats supplied
		formats = []string{"%s"}
	}

	var result interface{}
	var ok bool
	tried := "" // accumulates patterns tried, for possible error reporting

Found:
	for _, name := range nameVariants(baseName) {
		for _, pat := range formats {
			key := fmt.Sprintf(pat, name)
			result, ok = from[key]
			if ok {
				break Found
			}
			tried += key + "|"
		}
	}

	if !ok {
		return nil, fmt.Errorf("[%s] not found in map", tried[:len(tried)-1])
	}

	return result, nil
}

var uppersRE = regexp.MustCompile(`[[:upper:]]`)

func nameVariants(base string) []string {
	hyphens := strings.TrimLeft(uppersRE.ReplaceAllStringFunc(base, func(ch string) string {
		return "-" + ch
	}), "-")
	unders := strings.TrimLeft(uppersRE.ReplaceAllStringFunc(base, func(ch string) string {
		return "_" + ch
	}), "_")
	allM := map[string]struct{}{
		base: struct{}{},
		strings.ToLower(base): struct{}{},
		hyphens:               struct{}{},
		unders:                struct{}{},
		strings.ToLower(hyphens): struct{}{},
		strings.ToLower(unders):  struct{}{},
	}
	var all []string
	for n := range allM {
		all = append(all, n)
	}
	return all
}

// getBytes parses strings of the format '1.2G' and interprets a kB, MB,
// GB etc.
func getBytes(s string, err error) (int64, error) {
	const (
		b = 1
		k = b << 10
		m = k << 10
		g = m << 10
		t = g << 10
	)
	var mult int64
	switch strings.ToUpper(string(s[len(s)-1])) {
	case "B":
		mult = b
	case "K":
		mult = k
	case "M":
		mult = m
	case "G":
		mult = g
	case "T":
		mult = t
	default:
		return 0, err
	}
	ival, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		fval, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, err
		}
		return int64(fval * float64(mult)), nil
	}
	return ival * mult, nil
}
