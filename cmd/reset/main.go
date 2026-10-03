package main

import (
	"flag"
	"log"
)

func main() {
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	if err := run(root); err != nil {
		log.Fatal(err)
	}
}
