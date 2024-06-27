package main

import (
	"os"
)

func main() {
	testConfigFileName := os.Args[1]
	runTest(testConfigFileName)
}
