package main

import (
	"fmt"
	"unsafe"
)

func main() {
	slice1 := []string{"zhangsan", "lisi", "wanger", "zhaoliu"}
	slice2 := make([]int, 3)
	slice2 = [3,4,5]
	fmt.Println(len(slice1))
	fmt.Println(unsafe.Sizeof(slice1))
	fmt.Println(unsafe.Sizeof(slice2))
}
