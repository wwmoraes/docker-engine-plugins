package main

import (
	"fmt"
	"log"
	"os"

	"github.com/docker/go-plugins-helpers/volume"
)

func main() {
	driver, err := NewDriver()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	handler := volume.NewHandler(driver)
	if err := handler.ServeUnix("smbfs", 0); err != nil {
		log.Fatal(err)
	}
}
