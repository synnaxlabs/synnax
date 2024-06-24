package main

import (
	"bytes"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"os/exec"
	"strconv"
)

type DeleteParams struct {
	TimeRange telem.TimeRange `json:"time_range"`
	Channels  []string        `json:"channels"`
}

func (p DeleteParams) Serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.FormatInt(int64(p.TimeRange.Start), 10),
		strconv.FormatInt(int64(p.TimeRange.End), 10),
		strconv.Itoa(len(p.Channels)),
	)

	for _, g := range p.Channels {
		args = append(args, g)
	}

	return args
}

var _ NodeParams = &StreamParams{}

func deletePython(p NodeParams, identifier string) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := append([]string{"run", "python", "delete.py", identifier}, p.Serialize()...)
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
