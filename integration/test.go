package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"golang.org/x/sync/semaphore"
)

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

func runTest(testConfigFile string) (exitCode int) {
	ctx, cancel := context.WithCancel(context.Background())
	test := readTestConfig(testConfigFile)
	writeTestStart("timing.log", testConfigFile, test.Cluster, test.Setup)

	err := startCluster(ctx, test.Cluster)
	if err != nil {
		panic(err)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PANIC RECOVERED FOR CLEANUP FROM ERROR\n-----\n%s\n------\n", r)
			exitCode = 1
		}
		if err := runCleanUp(test.Cleanup); err != nil {
			panic(err)
		}

		cancel()
	}()

	if err := runSetUp(test.Setup); err != nil {
		panic(err)
	}

	for i, step := range test.Steps {
		if err := runStep(i, step); err != nil {
			panic(err)
		}
	}

	return
}

func runStep(i int, step TestStep) error {
	var (
		sem     = semaphore.NewWeighted(int64(len(step)))
		sCtx, _ = signal.Isolated()
	)
	fmt.Printf("--step %d\n", i)
	for n, node := range step {
		fmt.Printf("----node %d: %v with %s\n", n, node.Op, node.Client)

		if ok := sem.TryAcquire(1); !ok {
			panic("cannot acquire on semaphore")
		}

		n, node := n, node
		sCtx.Go(func(ctx context.Context) error {
			defer func() { sem.Release(1) }()
			time.Sleep((telem.TimeSpan(n * 10) * telem.Millisecond).Duration())
			if err := runNode(ctx, node, fmt.Sprintf("%d-%d", i, n)); err != nil {
				return err
			}

			return nil
		}, signal.CancelOnExitErr())
	}

	return sCtx.Wait()
}

func runNode(ctx context.Context, node TestNode, identifier string) error {
	var (
		stdErr, stdOut bytes.Buffer
		cmd            *exec.Cmd
		process        = make(chan error, 1)
		dir            string
	)

	switch node.Client {
	case "py":
		cmd = exec.Command("sh", "-c", node.Params.ToPythonCommand(identifier))
		dir = "./py"
	case "ts":
		cmd = exec.Command("sh", "-c", node.Params.ToTSCommand(identifier))
		dir = "./ts/src"
	default:
		return errors.Newf("Unrecognized client in %s: %s", identifier, node.Client)
	}

	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut
	cmd.Dir = dir
	cmd.Env = os.Environ()

	err := cmd.Start()
	if err != nil {
		return errors.Wrapf(
			err,
			"stdout: %s\nstderr: %s\n",
			stdOut.String(),
			stdErr.String(),
		)
	}

	go func() { process <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		fmt.Printf("----%s canceled\n", identifier)
		return cmd.Process.Kill()
	case err := <-process:
		if err != nil {
			fmt.Printf("----%s errored\n", identifier)
			return errors.Wrapf(
				err,
				"error in %s:\nstdout: %s\nstderr: %s\n",
				identifier,
				stdOut.String(),
				stdErr.String(),
			)
		}
		fmt.Printf("----%s finished\n", identifier)
		return nil
	}
}

func writeTestStart(fileName string, testFileName string, c ClusterParam, s SetUpParam) {
	var (
		fs     = xfs.Default
		f, err = fs.Open(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND)
	)

	if err != nil {
		panic(err)
	}

	msg := fmt.Sprintf(
		`-----Test Started | %s | %s-----
Configuration:
	TLS: %t
	Mem-based FS: %t
	Number of index channels: %d
	Number of data channels: %d
`,
		time.Now().Format(time.RFC3339),
		testFileName,
		!c.Insecure,
		c.MemFS,
		s.IndexChannels,
		s.DataChannels,
	)
	_, err = f.Write([]byte(msg))
	if err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}
