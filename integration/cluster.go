// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"bytes"
	"fmt"
	"github.com/synnaxlabs/x/errors"
	"net/http"
	"os/exec"
	"time"

	"github.com/synnaxlabs/x/telem"
)

type ClusterParam struct {
	Insecure bool `json:"insecure"`
	MemFS    bool `json:"mem_fs"`
}

func startCluster(p ClusterParam) (error, *bytes.Buffer, *bytes.Buffer, func() error) {
	if p == (ClusterParam{}) {
		fmt.Printf("--cannot find cluster startup configration, skipping\n")
		return nil, nil, nil, func() error { return nil }
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
		return err, &stdErr, &stdOut, func() error { return nil }
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrapf(
			err,
			"error in starting cluster.\nstdout: %s\nstderr: %s\n",
			stdOut.String(),
			stdErr.String(),
		), &stdOut, &stdErr, func() error { return nil }
	}

	time.Sleep(5 * telem.Second.Duration())
	client := &http.Client{}
	resp, err := client.Post("http://localhost:9090/api/v1/connectivity/check", "", nil)
	if err != nil {
		return errors.Wrap(err, "server did not start properly"),
			&stdOut,
			&stdErr,
			func() error { return nil }
	}
	defer resp.Body.Close()

	return nil, &stdOut, &stdErr, func() (err error) {
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
