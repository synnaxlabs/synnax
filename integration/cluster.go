package main

import (
	"bytes"
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

// startCluster starts a cluster that is sent a SIG_KILL when ctx is canceled.
func startCluster(p ClusterParam) (error, func() error) {
	if p == (ClusterParam{}) {
		fmt.Printf("--cannot find cluster startup configration, skipping\n")
		return nil, func() error { return nil }
	}

	fmt.Printf("--starting cluster\n")
	args := []string{"start", "-v"}
	if p.Insecure {
		args = append(args, "-i")
	}
	if p.MemFS {
		args = append(args, "-m")
	}

	var (
		stdOut, stdErr = bytes.Buffer{}, bytes.Buffer{}
		cmd            = exec.Command("./bin/synnax.exe", args...)
	)

	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut
	stdIn, err := cmd.StdinPipe()
	if err != nil {
		return err, func() error { return nil }
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrapf(
			err,
			"error in starting cluster.\nstdout: %s\nstderr: %s\n",
			stdOut.String(),
			stdErr.String(),
		), func() error { return nil }
	}

	time.Sleep(5 * telem.Second.Duration())

	return nil, func() (err error) {
		const stopKeyword = "stop"
		if _, err := stdIn.Write([]byte(stopKeyword)); err != nil {
			return err
		}
		if err = stdIn.Close(); err != nil {
			return err
		}
		return cmd.Wait()
	}
}
