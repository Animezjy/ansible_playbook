package main

import (
	"fmt"
)

type Animal struct {
	name   string
	weight float32
	age    int
}

type myint int8

func (animal *Animal) Eat() bool {
	if animal.age > 10 {
		fmt.Println("年龄太大了，少吃点")
		return false
	} else {
		fmt.Println("可劲造")
	}
	return true
}

func (this *Animal) ChangeAnimalName(animalName string) {
	this.name = animalName
}

func (this *Animal) PrintAnimalInfo() {
	fmt.Println(this.name)
	fmt.Println(this.age)
	fmt.Println(this.weight)
}

func main() {
	var animal Animal
	var num myint
	num = 33
	animal.name = "panda"
	animal.weight = 70.8
	animal.age = 19
	animal.Eat()
	animal.ChangeAnimalName("")
	animal.PrintAnimalInfo()
	fmt.Printf("num value =%d\n", num)
}
