package main

import (
	"io/ioutil"
	"os"
	"runtime"

	"github.com/gwaycc/supd"
)

func main() {
	// Get root auth for security
	if runtime.GOOS != "windows" {
		rootAuthFile := "/var/supd.auth.lock"
		if err := ioutil.WriteFile(rootAuthFile, []byte{}, 0666); err != nil {
			panic(err)
		}
		// pass
		if err := os.Remove(rootAuthFile); err != nil {
			panic(err)
		}
	}

	// run command
	supd.Run()
}
