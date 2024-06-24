package main

import (
	"bytes"
	"github.com/synnaxlabs/x/errors"
	"os/exec"
	"strconv"
)

type StreamParams struct {
	StartTimeStamp   int      `json:"start_time_stamp"`
	CloseAfterFrames int      `json:"close_after_frames"`
	Channels         []string `json:"channels"`
}

func (p StreamParams) Serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.FormatInt(int64(p.StartTimeStamp), 10),
		strconv.Itoa(p.CloseAfterFrames),
		strconv.Itoa(len(p.Channels)),
	)

	for _, g := range p.Channels {
		args = append(args, g)
	}

	return args
}

var _ NodeParams = &StreamParams{}

func streamPython(p NodeParams, identifier string) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := append([]string{"run", "python", "stream.py", identifier}, p.Serialize()...)
	cmd := exec.Command("poetry", args...)
	cmd.Dir = "./py"
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "stdout: %s\nstderr: %s\n", stdout.String(), stderr.String())
	}

	return nil
}
