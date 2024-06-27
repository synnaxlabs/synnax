package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

type CleanUpParam struct {
	DeleteAllChannels bool   `json:"delete_all_channels"`
	Client            string `json:"client"`
}

func runCleanUp(param CleanUpParam) error {
	if param == (CleanUpParam{}) {
		fmt.Printf("--cannot find cleanup configuration, skipping\n")
		return nil
	}

	fmt.Printf("--cleaning up\n")
	switch param.Client {
	case "py":
		return cleanUpPython(param)
	default:
		panic("unrecognized client in cleanup")
	}
}

func cleanUpPython(param CleanUpParam) error {
	if !param.DeleteAllChannels {
		return nil
	}
	args := "-c poetry install && poetry run python delete_all.py"
	var (
		cmd            = exec.Command("sh", strings.Split(args, " ")...)
		stdErr, stdOut bytes.Buffer
	)

	cmd.Dir = "./py"
	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut

	if err := cmd.Run(); err != nil {
		return errors.Newf("err: %s\nstderr: %s\nstdout: %s", err.Error(), stdErr.String(), stdOut.String())
	}
	return nil
}
