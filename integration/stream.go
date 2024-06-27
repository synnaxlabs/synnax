package main

import (
	"strconv"
	"strings"
)

type StreamParams struct {
	StartTimeStamp   int      `json:"start_time_stamp"`
	CloseAfterFrames int      `json:"close_after_frames"`
	Channels         []string `json:"channels"`
}

func (p StreamParams) serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.FormatInt(int64(p.StartTimeStamp), 10),
		strconv.Itoa(p.CloseAfterFrames),
		strconv.Itoa(len(p.Channels)),
	)

	for _, g := range p.Channels {
		args = append(args, g)
	}

	return args
}

func (p StreamParams) ToPythonCommand(identifier string) string {
	cmd := "poetry run python stream.py "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

func (p StreamParams) ToTSCommand(identifier string) string {
	cmd := "npx tsx stream.ts "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

var _ NodeParams = &StreamParams{}
