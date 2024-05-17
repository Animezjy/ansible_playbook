package main

import (
	"GoServe/global"
	"log"
)

func init(){
err := setupDB()
	if err != nil {
		log.Fatalf("init.setupDBEngine err: %v", err)
	}
}

func main(){
    r := gin.Default()
    r.Run(":9090")
}

func setupDB() error {
    err := global.MongodbJoin()
    if err != nil {
		return err
	}
	return nil
}
