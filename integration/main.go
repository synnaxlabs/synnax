package main

import (
	"fmt"
)

type Operation int

const (
	Read Operation = iota + 1
	Write
	Delete
	Stream
)

type ChannelGroup struct {
	indexChannels []string
	dataChannels  []string
}

type NodeParams struct {
	numWriters       int
	domains          int
	samplesPerDomain int
	channelGroups    []ChannelGroup
}

type TestNode struct {
	op     Operation
	client string
	params NodeParams
}

type TestStep = []TestNode

var testSequence = []TestStep{
	[]TestNode{
		{
			op:     Write,
			client: "py",
			params: NodeParams{
				channelGroups:    []ChannelGroup{{indexChannels: []string{"0"}, dataChannels: []string{"0-1"}}},
				numWriters:       1,
				domains:          1,
				samplesPerDomain: 1,
			},
		},
		{
			op:     Stream,
			client: "cpp",
			params: NodeParams{},
		},
	},
}

func runNode(node TestNode) {
	switch node.op {
	case Read:
		break
	case Write:
		break
	case Stream:
		break
	case Delete:
		break
	}
}

func main() {
	fmt.Println("Setting up")
	err := setUp(SetUpParam{
		indexChannels: 1,
		datachannels:  1,
	})
	if err != nil {
		panic(err)
	}
	for i, seq := range testSequence {
		fmt.Printf("Testing step %s", seq)
		for _, node := range seq {
			if err = writePython(testSequence[0][0].params); err != nil {
				panic(err)
			}
		}
	}
}
