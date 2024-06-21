package main

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/x/signal"

	"github.com/synnaxlabs/x/telem"
	"golang.org/x/sync/semaphore"
)

type Operation int

const (
	Read Operation = iota + 1
	Write
	Delete
	Stream
)

func (o Operation) String() string {
	switch o {
	case Read:
		return "read"
	case Write:
		return "write"
	case Delete:
		return "delete"
	case Stream:
		return "stream"
	}
	return "unknown"
}

type ChannelGroup struct {
	indexChannels []string
	dataChannels  []string
}

type NodeParams interface {
	Serialize() []string
}

type TestNode struct {
	op     Operation
	client string
	params NodeParams
}

type TestStep = []TestNode

var testSequence = []TestStep{
	{
		{
			op:     Write,
			client: "py",
			params: WriteParams{
				channelGroups:    []ChannelGroup{{indexChannels: []string{"0"}, dataChannels: []string{"0-0"}}},
				numWriters:       1,
				domains:          100,
				samplesPerDomain: 1000000,
				timeRange:        telem.TimeStamp(0).SpanRange(100 * telem.Second),
			},
		},
		{
			op:     Write,
			client: "py",
			params: WriteParams{
				channelGroups:    []ChannelGroup{{indexChannels: []string{"1"}, dataChannels: []string{"1-0"}}},
				numWriters:       1,
				domains:          100,
				samplesPerDomain: 1000,
				timeRange:        telem.TimeStamp(0).SpanRange(100 * telem.Second),
			},
		},
		{
			op:     Stream,
			client: "py",
			params: StreamParams{
				channels:         []string{"1-0", "0-0"},
				closeAfterFrames: 100,
			},
		},
	},
	{
		{
			op:     Delete,
			client: "py",
			params: DeleteParams{
				tr:       (50 * telem.SecondTS).Range(75 * telem.SecondTS),
				channels: []string{"0-0", "1-0"},
			},
		},
		{
			op:     Write,
			client: "py",
			params: WriteParams{
				channelGroups: []ChannelGroup{
					{indexChannels: []string{"0"}, dataChannels: []string{"0-0"}},
					{indexChannels: []string{"1"}, dataChannels: []string{"1-0"}},
				},
				numWriters:       2,
				domains:          50,
				samplesPerDomain: 1000,
				timeRange:        (120 * telem.SecondTS).SpanRange(50 * telem.Second),
			},
		},
		{
			op:     Stream,
			client: "py",
			params: StreamParams{
				channels:         []string{"1-0", "0-0", "1", "0"},
				closeAfterFrames: 1,
			},
		},
	},
}

func runNode(node TestNode) error {
	switch node.op {
	case Read:
		break
	case Write:
		switch node.client {
		case "py":
			return writePython(node.params)
		}
		break
	case Stream:
		switch node.client {
		case "py":
			return streamPython(node.params)
		}
		break
	case Delete:
		switch node.client {
		case "py":
			return deletePython(node.params)
		}
		break
	}

	return nil
}

func runStep(step TestStep) error {
	var (
		sem     = semaphore.NewWeighted(int64(len(step)))
		sCtx, _ = signal.Isolated()
	)
	for i, node := range step {
		fmt.Printf("----node %d: %v with %s\n", i, node.op, node.client)
		if ok := sem.TryAcquire(1); !ok {
			panic("cannot acquire on semaphore")
		}

		i, node := i, node
		sCtx.Go(func(ctx context.Context) error {
			defer func() {
				sem.Release(1)
				fmt.Printf("----finished node %d\n", i)
			}()
			err := runNode(node)
			if err != nil {
				fmt.Printf("error in node %d: %s", i, err.Error())
			}
			return err
		})
	}

	return sCtx.Wait()
}

func main() {
	fmt.Print("Setting up\n")
	err := setUp(SetUpParam{
		indexChannels: 2,
		datachannels:  2,
	})
	if err != nil {
		panic(err)
	}
	for i, step := range testSequence {
		fmt.Printf("--step %d\n", i)
		if err = runStep(step); err != nil {
			panic(err)
		}
	}
}
