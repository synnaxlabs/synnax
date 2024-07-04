package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type ClusterParam struct {
	Insecure bool `json:"insecure"`
	MemFS    bool `json:"mem_fs"`
}

func startCluster(ctx context.Context, p ClusterParam) (error, func() error) {
	if p == (ClusterParam{}) {
		fmt.Printf("--cannot find cluster startup configration, skipping\n")
		return nil, func() error { return nil }
	}

	fmt.Printf("--starting cluster\n")
	args := []string{"run", "main.go", "start", "-d", "../integration/synnax-data"}
	if p.Insecure {
		args = append(args, "-i")
	}
	if p.MemFS {
		args = append(args, "-m")
	}

	var (
		stdOut, stdErr = bytes.Buffer{}, bytes.Buffer{}
		cmd            = exec.CommandContext(ctx, "go", args...)
	)

	cmd.Dir = "./../synnax"
	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut

	err := cmd.Start()
	if err != nil {
		return errors.Newf(
			"error in starting cluster.\nstdout: %s\nstderr: %s\n",
			stdOut.String(),
			stdErr.String(),
		), func() error { return nil }
	}

	time.Sleep(5 * telem.Second.Duration())
	return nil, func() error {
		// FIXME: This only kills the cluster process and not any processes it started.
		return cmd.Process.Kill()
	}
}
