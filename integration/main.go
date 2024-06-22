package main

import "os"

func main() {
	if len(os.Args) != 2 {
		panic("usage:  go run . [name of test configuration file]")
	}

	testConfigFileName := os.Args[1]
	runTest(testConfigFileName)
}
