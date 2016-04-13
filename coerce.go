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
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

// Coerce attempts to unmarshall the values in 'from' into the fields
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
//	err := coerce.Coerce(&myx, mymap, "--%s", "-%s")
//	fmt.Println(err, myx) // <nil> {[5 12 512] true hello}
//
// Note: coercing unexported fields uses 'unsafe' pointers
//
func Coerce(to interface{}, from map[string]interface{}, formats ...string) error {

	// parse errors are accumulated into errstr
	errstr := ""

	// get target as reflect.Value and check kind:
	pt := reflect.ValueOf(to)
	vt := reflect.Indirect(pt)
	if vt.Kind() != reflect.Struct || pt.Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("Cast: expected *struct for 'to', got %v", pt.Kind()))
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
				errstr += "Coerce: !CanSet() field " + f.Name + "\n"
				continue
			}
		}

		// look for field name in map keys
		v, err := findVal(f.Name, from, formats)
		if err != nil {
			errstr += err.Error() + "\n"
			continue
		}

		if v == nil {
			// nil value in map - set field to its type's zero value
			vf.Set(reflect.Zero(vf.Type()))
			continue
		}

		vv := reflect.ValueOf(v)

		// try for direct assign:
		if reflect.TypeOf(vv).AssignableTo(f.Type) {
			vf.Set(vv)
			continue
		}

		// unmarshall from a single value:
		if vv.Kind() != reflect.Slice {
			err := unmarshall(vf, vv)
			if err != nil {
				errstr += err.Error() + "\n"
			}
			continue
		}

		// unmarshall from a slice...:
		if vf.Kind() == reflect.Slice {
			// ...to a slice:
			// set slice size:
			vf.Set(reflect.MakeSlice(vf.Type(), vv.Len(), vv.Len()))

			for j := 0; j < vv.Len(); j++ {
				// unmarshall slice elements
				err := unmarshall(vf.Index(j), vv.Index(j))
				if err != nil {
					errstr += err.Error() + "\n"
				}
			}

		} else if vv.Len() == 1 {
			// tolerate mapping of slices with length==1 to a single field
			err := unmarshall(vf, vv.Index(0))
			if err != nil {
				errstr += err.Error() + "\n"
			}
		} else {
			errstr += "Coerce: can't coerce " + f.Name + " from multi-value slice\n"
		}

	}

	if errstr != "" {
		return errors.New(errstr[:len(errstr)-1]) // strips trailling newline
	}
	return nil
}

// Unmarshal tries to set value of 'to' based on content of 'from'
func Unmarshall(to interface{}, from interface{}) error {
	p := reflect.ValueOf(to)
	if p.Kind() != reflect.Ptr {
		return errors.New("Unmarshall: 'to' must be pointer")
	}
	return unmarshall(reflect.Indirect(p), reflect.ValueOf(from))
}

// unmarshal tries to set value of 'to' based on content of 'from'
func unmarshall(to reflect.Value, from reflect.Value) error {

	// try for direct assign:
	if from.Type().AssignableTo(to.Type()) {
		to.Set(from)
		return nil
	}

	// do it the hard way:
	switch from.Kind() {

	case reflect.String:

		s := from.String()

		switch to.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ival, err := strconv.ParseInt(s, 10, to.Type().Bits())
			if err != nil {
				// try again looking for B/K/M/G/T
				ival, err = getBytes(s, err)
				if err != nil {
					return err
				}
			}
			to.SetInt(ival)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uval, err := strconv.ParseUint(s, 10, to.Type().Bits())
			if err != nil {
				// try again looking for B/K/M/G/T
				ival, e := getBytes(s, err)
				if e != nil {
					return e
				}
				uval = uint64(ival)
			}
			to.SetUint(uval)
		case reflect.Float32, reflect.Float64:
			fval, err := strconv.ParseFloat(s, to.Type().Bits())
			if err != nil {
				return err
			}
			to.SetFloat(fval)
		default:
			return errors.New(fmt.Sprintf("Unhandled destination kind: %v\n", to.Kind()))
		}
	default:
		return errors.New(fmt.Sprintf("Don't know how to unmarshall %v to %v\n",
			from.Kind(),
			to.Kind()))
	}
	return nil
}

// findVal tries to find map key matching field name formatted as per formats
func findVal(name string, from map[string]interface{}, formats []string) (interface{}, error) {

	if len(formats) == 0 {
		// handle case where no formats supplied
		formats = []string{"%s"}
	}

	var result interface{}
	var ok bool
	tried := "" // accumulates patterns tried, for possible error reporting
	for _, pat := range formats {
		key := fmt.Sprintf(pat, name)
		result, ok = from[key]
		if ok {
			break
		}
		tried += key + "|"
	}

	if !ok {
		return nil, errors.New("Coerce: [" + tried[:len(tried)-2] + "] not found in map")
	}

	return result, nil
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
