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

// startcluster starts a cluster that is sent a SIG_KILL when ctx is canceled.
func startCluster(p ClusterParam) (error, func() error) {
	if p == (ClusterParam{}) {
		fmt.Printf("--cannot find cluster startup configration, skipping\n")
		return nil, func() error { return nil }
	}

	fmt.Printf("--starting cluster\n")
	args := []string{"start", "-v", "--debug"}
	if p.Insecure {
		args = append(args, "-i")
	}
	if p.MemFS {
		args = append(args, "-m")
	}

	var (
		stdOut, stdErr = bytes.Buffer{}, bytes.Buffer{}
		cmd            = exec.Command("./bin/synnax", args...)
		pgoCmd         *exec.Cmd
	)

	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut

	err := cmd.Start()
	if err != nil {
		return errors.Wrapf(
			err,
			"error in starting cluster.\nstdout: %s\nstderr: %s\n",
			stdOut.String(),
			stdErr.String(),
		), func() error { return nil }
	}

	time.Sleep(5 * telem.Second.Duration())

	pgoCmd = startPGO()
	if err := pgoCmd.Start(); err != nil {
		return errors.Wrap(err, "error in starting PGO"), func() error { return nil }
	}

	return nil, func() (err error) {
		if pgoCmd.ProcessState != nil && !pgoCmd.ProcessState.Exited() {
			err = pgoCmd.Wait()
		}

		return errors.CombineErrors(err, cmd.Process.Kill())
	}
}

func startPGO() *exec.Cmd {
	// It is important to note that every test must run for > 60s for PGO to have enough
	// time to gather the profile.
	url := "http://localhost:9090/debug/pprof/profile?seconds=60"
	return exec.Command("curl", "-o", "../synnax/default.pgo", url)
}
