package main

import (
	"fmt"
	"os"
)

func assert(err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "main: %s\n", err)
	os.Exit(1)
}

func main() {
	fd, err := os.Open("/proc/self/mounts")
	assert(err)
	defer fd.Close()

	mounts, err := NewMountsCache(fd)
	assert(err)

	fmt.Printf("%#+v\n", mounts)

	// driver, err := NewDriver()
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }

	// handler := volume.NewHandler(driver)
	// if err := handler.ServeUnix("cifs", 0); err != nil {
	// 	log.Fatal(err)
	// }
}
