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

func (p DeleteParams) ToPythonCommand(identifier string) []string {
	cmd := "-c poetry install && poetry run python delete.py " + identifier
	return append(strings.Split(cmd, " "), p.serialize()...)
}

func (p DeleteParams) ToTSCommand(identifier string) []string {
	cmd := "-c npx tsx delete.ts " + identifier
	return append(strings.Split(cmd, " "), p.serialize()...)
}

var _ NodeParams = &DeleteParams{}
