package main

import "fmt"

func main() {
	num1 := 100
	switch num1 {
	case 98, 99:
		fmt.Println("Its equal to 98")
	case 100:
		fmt.Println("Its equal to 100")
	default:
		fmt.Println("It's not equal to 98 or 100")
	}
}
