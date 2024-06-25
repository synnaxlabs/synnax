package main

import (
	"bytes"
	"os/exec"
	"strconv"

	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/x/errors"
)

type ReadParams struct {
	NumIterators  int             `json:"num_iterators"`
	ChunkSize     int             `json:"chunk_size"`
	Bounds        telem.TimeRange `json:"bounds"`
	ChannelGroups [][]string      `json:"channel_groups"`
}

func (p ReadParams) Serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.Itoa(p.NumIterators),
		strconv.Itoa(p.ChunkSize),
		strconv.FormatInt(int64(p.Bounds.Start), 10),
		strconv.FormatInt(int64(p.Bounds.End), 10),
		strconv.Itoa(len(p.ChannelGroups)),
	)

	for _, g := range p.ChannelGroups {
		args = append(args, strconv.Itoa(len(g)))
		args = append(args, g...)
	}

	return args
}

var _ NodeParams = &WriteParams{}

func readPython(p NodeParams, identifier string) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := append([]string{"run", "python", "read.py", identifier}, p.Serialize()...)
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

func readTS(p NodeParams, identifier string) error {
	args := append([]string{"tsx", "read.ts", identifier}, p.Serialize()...)
	cmd := exec.Command("npx", args...)
	cmd.Dir = "./ts"
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "stdout: %s\nstderr: %s\n", stdout.String(), stderr.String())
	}

	return nil
}
