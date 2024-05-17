package main

import "fmt"

type Config struct {
	title string
	tag   string
	value string
}

func (c *Config) SetTitle(name string, value string) {
	c.title = name
	c.value = c.value
	if c.title == "" {
		c.tag = "None"
	} else {
		c.tag = c.title
	}
}

func main() {
	fmt.Println("hello")
	c := Config{"Go", "33"}
}
