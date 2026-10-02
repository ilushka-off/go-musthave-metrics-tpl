package main

import stdos "os"

func main() {
	stdos.Exit(1) // want "direct call to os.Exit in main function of package main is forbidden"
}
