# coerce
coerce is a golang package which coerces map[string]interface{} values into struct fields

# Installation
```
go get github.com/SeeSpotRun/coerce
```

# Example

```
package main
import (
	"fmt"
	"github.com/SeeSpotRun/coerce"
)

func main() {
	type x struct{
		intslice  []int
		boolval   bool
		s         string
	}

	mymap := map[string]interface{} {
		"intslice":  []string {"5", "12", "0.5k"},
		"boolval" :  true,
		"s"       :  "hello",
	}

	var myx x

	err := coerce.Coerce(&myx, mymap)
	fmt.Println(err, myx) // gives: <nil> {[5 12 512] true hello}

}
```
