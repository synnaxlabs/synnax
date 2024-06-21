package main

import (
	"bytes"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"os/exec"
	"strconv"
)

type DeleteParams struct {
	tr       telem.TimeRange
	channels []string
}

func (p DeleteParams) Serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.FormatInt(int64(p.tr.Start), 10),
		strconv.FormatInt(int64(p.tr.End), 10),
		strconv.Itoa(len(p.channels)),
	)

	for _, g := range p.channels {
		args = append(args, g)
	}

	return args
}

var _ NodeParams = &StreamParams{}

func deletePython(p NodeParams) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := append([]string{"run", "python", "delete.py"}, p.Serialize()...)
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
