package main

import (
	"bytes"
	"os/exec"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/x/errors"
)

type ChannelGroup struct {
	IndexChannels []string `json:"index_channels"`
	DataChannels  []string `json:"data_channels"`
}

type WriteParams struct {
	NumWriters           int             `json:"num_writers"`
	Domains              int             `json:"domains"`
	SamplesPerDomain     int             `json:"samples_per_domain"`
	TimeRange            telem.TimeRange `json:"time_range"`
	AutoCommit           bool            `json:"auto_commit"`
	IndexPersistInterval telem.TimeSpan  `json:"index_persist_interval"`
	WriterMode           writer.Mode     `json:"writer_mode"`
	ChannelGroups        []ChannelGroup  `json:"channel_groups"`
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
		strconv.FormatBool(p.AutoCommit),
		strconv.FormatInt(int64(p.IndexPersistInterval), 10),
		strconv.Itoa(int(p.WriterMode)),
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

func writePython(p NodeParams, identifier string) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := append([]string{"run", "python", "write.py", identifier}, p.Serialize()...)
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

func writeTS(p NodeParams, identifier string) error {
	args := append([]string{"tsx", "write.ts", identifier}, p.Serialize()...)
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