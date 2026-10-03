package main

import "os"

func main() {
	os.Exit(1) // want "direct call to os.Exit in main function of package main is forbidden"
}

func helper() {
	// os.Exit outside of main must not be flagged.
	os.Exit(2)
}
