package main

import (
	"context"
	"fmt"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"golang.org/x/sync/semaphore"
	"os"
)

func runNode(node TestNode) error {
	switch node.Op {
	case Read:
		switch node.Client {
		case "py":
			return readPython(node.Params)
		}
		break
	case Write:
		switch node.Client {
		case "py":
			return writePython(node.Params)
		}
		break
	case Stream:
		switch node.Client {
		case "py":
			return streamPython(node.Params)
		}
		break
	case Delete:
		switch node.Client {
		case "py":
			return deletePython(node.Params)
		}
		break
	}

	return nil
}

func runStep(i int, step TestStep) error {
	var (
		sem     = semaphore.NewWeighted(int64(len(step)))
		sCtx, _ = signal.Isolated()
	)
	fmt.Printf("--step %d\n", i)
	for i, node := range step {
		fmt.Printf("----node %d: %v with %s\n", i, node.Op, node.Client)
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
				fmt.Printf("----error in node %d: %s\n", i, err.Error())
			}

			return err
		})
	}

	return sCtx.Wait()
}

func readTestConfig(fileName string) TestSequence {
	fs := xfs.Default
	f, err := fs.Open(fileName, os.O_RDONLY)
	if err != nil {
		panic(err)
	}
	s, err := f.Stat()
	if err != nil {
		panic(err)
	}
	b := make([]byte, s.Size())
	_, err = f.Read(b)
	if err != nil {
		panic(err)
	}
	seq, err := UnmarshalJSON(b)
	if err != nil {
		panic(err)
	}
	return seq
}

func runTest(testConfigFile string) {
	test := readTestConfig(testConfigFile)

	endCommand := startCluster(test.Cluster)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PANIC RECOVERED FOR CLEANUP\n")
		}
		if err := runCleanUp(test.Cleanup); err != nil {
			panic(err)
		}

		if err := endCommand(); err != nil {
			panic(err)
		}
	}()

	if err := runSetUp(test.Setup); err != nil {
		panic(err)
	}

	for i, step := range test.Body {
		if err := runStep(i, step); err != nil {
			panic(err)
		}
	}
}
