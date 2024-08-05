package main

import (
	"flag"
	"fmt"
	"os"
)

var verboseFlag = flag.Bool("v", false, "verbose")

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: program [-v] <configfile>")
		os.Exit(1)
	}

	testConfigFileName := flag.Arg(0)

	exitCode := runTest(testConfigFileName, *verboseFlag)
	os.Exit(exitCode)
}