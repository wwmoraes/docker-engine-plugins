package main

import (
	"log"
)

func main() {
	handler := NewHandler()
	if err := handler.ServeUnix("cifs", 0); err != nil {
		log.Fatal(err)
	}
}
