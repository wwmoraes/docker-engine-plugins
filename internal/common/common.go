package common

import (
	"log"
	"os"
)

var errLog = log.New(os.Stderr, "", log.LstdFlags)

func Assert(err error) {
	if err == nil {
		return
	}

	errLog.Fatalln(err)
}
