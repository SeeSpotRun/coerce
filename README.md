# coerce
Golang typecasting for the lazy.

Includes coercion of map[string]interface{} values into named fields.

## Installation
```
go get github.com/SeeSpotRun/coerce
```

## API
See [godoc](https://godoc.org/github.com/SeeSpotRun/coerce)

## Example

```
package main
import (
	"fmt"
	"github.com/SeeSpotRun/coerce"
)

func main() {

	// cast to a single var:
	var i integer
	s := "1234"
	coerce.Var(&i, s) //      i == int(1234)


	// return a typecast var:
	f := coerce.Float32(s) // f == float32(1234.0)


	// coerce struct fields from a map[string]interface{}:
	type x struct{
		intslice  []int
		boolval   bool
		s         string
	}
	var myx x

	mymap := map[string]interface{} {
		"intslice":  []string {"5", "12", "0.5k"},
		"boolval" :  true,
		"s"       :  "hello",
	}

	err := coerce.Struct(&myx, mymap)
	fmt.Println(err, myx) // gives: <nil> {[5 12 512] true hello}

}
```

## Contributing
Feel welcome to contribute via Issues and Pull Requests

[![Go Report Card](https://goreportcard.com/badge/github.com/SeeSpotRun/coerce)](https://goreportcard.com/report/github.com/SeeSpotRun/coerce)
