package main

import (
	"os"
)

func main() {
	testConfigFileName := os.Args[1]
	exitCode := runTest(testConfigFileName)
	os.Exit(exitCode)
}
