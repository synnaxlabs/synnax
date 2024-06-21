package main

import (
	"bytes"
	"github.com/synnaxlabs/x/errors"
	"os/exec"
	"strconv"
)

type StreamParams struct {
	startTimeStamp   int
	closeAfterFrames int
	channels         []string
}

func (p StreamParams) Serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.FormatInt(int64(p.startTimeStamp), 10),
		strconv.Itoa(p.closeAfterFrames),
		strconv.Itoa(len(p.channels)),
	)

	for _, g := range p.channels {
		args = append(args, g)
	}

	return args
}

var _ NodeParams = &StreamParams{}

func streamPython(p NodeParams) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := append([]string{"run", "python", "stream.py"}, p.Serialize()...)
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
