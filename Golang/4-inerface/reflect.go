package main

type Config struct {
	title string `json: "name"`
	price string `json: "price"`
}

func (this *Config) GetConfig() {
	this.title = title
}

func main() {
}
