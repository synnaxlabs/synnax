// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"fmt"
	"github.com/synnaxlabs/x/calc"
)

type calcRunner struct {
	// construct to provide data for each opearand
	sources map[string]chan float64
	// construct to output transformed data
	sink chan float64
	// description of operations and operand
	rule string
}

func main() {
	fmt.Println("Hello, World!")
	e := calc.Expression{}
	err := e.Build("1 + 2")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(e.Evaluate(nil))
	}
}

func runCalc(c calcRunner) {
	// parse the rule into a list of operations and operands
	// for each operation, read the operands from the channels
	// perform the operation
	// write the result to the sink
}

// map of operands to channels to read streams from
// so the writing to the channel is done externally either with an iterator or a generator

// I want ot be able to use the interface given for calc
// however it has a resolver which takes in a map from a string to a single float64
// if i want tl be able to do that with alot of data
// i either make that a map from a string to a channel or a slice of data
// i am not sure what the best move is here
// its a matter of how slow it is
