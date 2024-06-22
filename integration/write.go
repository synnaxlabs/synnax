package main

import (
	"bytes"
	"github.com/synnaxlabs/x/telem"
	"os/exec"
	"strconv"

	"github.com/synnaxlabs/x/errors"
)

type ChannelGroup struct {
	IndexChannels []string `json:"index_channels"`
	DataChannels  []string `json:"data_channels"`
}

type WriteParams struct {
	NumWriters       int             `json:"num_writers"`
	Domains          int             `json:"domains"`
	SamplesPerDomain int             `json:"samples_per_domain"`
	TimeRange        telem.TimeRange `json:"time_range"`
	ChannelGroups    []ChannelGroup  `json:"channel_groups"`
}

func (p WriteParams) Serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.Itoa(p.NumWriters),
		strconv.Itoa(p.Domains),
		strconv.Itoa(p.SamplesPerDomain),
		strconv.FormatInt(int64(p.TimeRange.Start), 10),
		strconv.FormatInt(int64(p.TimeRange.End), 10),
		strconv.Itoa(len(p.ChannelGroups)),
	)

	for _, g := range p.ChannelGroups {
		args = append(
			args,
			strconv.Itoa(len(g.IndexChannels)),
			strconv.Itoa(len(g.DataChannels)),
		)
		args = append(args, g.IndexChannels...)
		args = append(args, g.DataChannels...)
	}

	return args
}

var _ NodeParams = &WriteParams{}

func writePython(p NodeParams) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := append([]string{"run", "python", "write.py"}, p.Serialize()...)
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
