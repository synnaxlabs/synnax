package main

import (
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/telem"
)

type DeleteParams struct {
	TimeRange telem.TimeRange `json:"time_range"`
	Channels  []string        `json:"channels"`
}

func (p DeleteParams) serialize() []string {
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

func (p DeleteParams) ToPythonCommand(identifier string) string {
	cmd := "poetry run python delete.py "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

func (p DeleteParams) ToTSCommand(identifier string) string {
	cmd := "npx tsx delete.ts "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

var _ NodeParams = &DeleteParams{}
