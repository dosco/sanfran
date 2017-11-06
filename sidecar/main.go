package main

import (
	"flag"

	"github.com/golang/glog"
)

const port = 8080

func main() {
	flag.Parse()
	defer glog.Flush()

	initServer(port)
}
