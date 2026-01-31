package duplication

import "fmt"

// BlockA contains an 8-line duplicated block.
func BlockA() {
	x := 1
	y := 2
	z := x + y
	fmt.Println(z)
	fmt.Println(x * y)
	fmt.Println(z * x)
	fmt.Println(y + z)
	fmt.Println(x + y + z)
}

// BlockB contains the same 8-line duplicated block.
func BlockB() {
	x := 1
	y := 2
	z := x + y
	fmt.Println(z)
	fmt.Println(x * y)
	fmt.Println(z * x)
	fmt.Println(y + z)
	fmt.Println(x + y + z)
}
