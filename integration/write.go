package main

import (
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"
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

func (p WriteParams) serialize() []string {
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

func (p WriteParams) ToPythonCommand(identifier string) string {
	cmd := "poetry run python write.py "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

func (p WriteParams) ToTSCommand(identifier string) string {
	cmd := "npx tsx write.ts "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

var _ NodeParams = &WriteParams{}
