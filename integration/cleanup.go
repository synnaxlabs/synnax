package main

import (
	"context"
	"fmt"
)

type CleanUpParam struct {
	DeleteAllChannels bool   `json:"delete_all_channels"`
	Client            string `json:"client"`
}

func (p CleanUpParam) serialize() []string {
	return []string{}
}

func (p CleanUpParam) ToPythonCommand(_ string) string {
	if p == (CleanUpParam{}) {
		return ""
	}

	return "poetry run python delete_all.py"
}

func (p CleanUpParam) ToTSCommand(_ string) string {
	panic("unimplemented")
}

var _ NodeParams = &CleanUpParam{}

func runCleanUp(param CleanUpParam) error {
	fmt.Printf("--cleaning up\n")

	if param.DeleteAllChannels {
		switch param.Client {
		case "py":
			return testPython(context.Background(), param, "cleanup")
		case "ts":
			return testTS(context.Background(), param, "cleanup")
		default:
			panic("unrecognized client in cleanup")
		}
	}

	return nil
}
