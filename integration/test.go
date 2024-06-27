package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"golang.org/x/sync/semaphore"
)

func runNode(node TestNode, identifier string) error {
	switch node.Client {
	case "py":
		return testPython(node.Params, identifier)
	case "ts":
		return testTS(node.Params, identifier)
	}

	return errors.Newf("unknown client for %s: %s on %s", identifier, node.Op, node.Client)
}

func runStep(i int, step TestStep) error {
	var (
		sem          = semaphore.NewWeighted(int64(len(step)))
		sCtx, cancel = signal.Isolated()
	)
	fmt.Printf("--step %d\n", i)
	for n, node := range step {
		fmt.Printf("----node %d: %v with %s\n", n, node.Op, node.Client)
		if ok := sem.TryAcquire(1); !ok {
			panic("cannot acquire on semaphore")
		}

		n, node := n, node
		sCtx.Go(func(ctx context.Context) error {
			defer func() {
				sem.Release(1)
				fmt.Printf("----finished node %d\n", n)
			}()
			err := runNode(node, fmt.Sprintf("%d-%d", i, n))
			if err != nil {
				cancel()
				return errors.Newf("----error in node %d: %s\n", n, err.Error())
			}

			return nil
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

	err, endCommand := startCluster(test.Cluster)
	if err != nil {
		panic(err)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PANIC RECOVERED FOR CLEANUP from error\n-----\n%s\n------\n", r)
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

func testPython(p NodeParams, identifier string) error {
	var (
		stdErr, stdOut bytes.Buffer
		cmd            = exec.Command("sh", p.ToPythonCommand(identifier)...)
	)
	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut
	cmd.Dir = "./py"

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(
			err,
			"stdout: %s\nstderr: %s\n",
			stdOut.String(),
			stdErr.String(),
		)
	}

	return nil
}

func testTS(p NodeParams, identifier string) error {
	var (
		stderr, stdout bytes.Buffer
		cmd            = exec.Command("sh", p.ToTSCommand(identifier)...)
	)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Dir = "./ts/src"

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err,
			"stdout: %s\nstderr: %s\n",
			stdout.String(),
			stderr.String(),
		)
	}

	return nil
}
