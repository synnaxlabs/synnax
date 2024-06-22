package main

import (
	"bytes"
	"github.com/synnaxlabs/x/telem"
	"os/exec"
	"strconv"

	"github.com/synnaxlabs/x/errors"
)

type ReadParams struct {
	numIterators  int
	chunkSize     int
	bounds        telem.TimeRange
	channelGroups [][]string
}

func (p ReadParams) Serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.Itoa(p.numIterators),
		strconv.Itoa(p.chunkSize),
		strconv.FormatInt(int64(p.bounds.Start), 10),
		strconv.FormatInt(int64(p.bounds.End), 10),
		strconv.Itoa(len(p.channelGroups)),
	)

	for _, g := range p.channelGroups {
		args = append(args, strconv.Itoa(len(g)))
		args = append(args, g...)
	}

	return args
}

var _ NodeParams = &WriteParams{}

func readPython(p NodeParams) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := append([]string{"run", "python", "read.py"}, p.Serialize()...)
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
