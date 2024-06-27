package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/synnaxlabs/x/errors"
)

type SetUpParam struct {
	IndexChannels int    `json:"index_channels"`
	DataChannels  int    `json:"data_channels"`
	Client        string `json:"client"`
}

func runSetUp(param SetUpParam) error {
	if param == (SetUpParam{}) {
		fmt.Printf("--cannot find setup configuration, skipping\n")
	}

	fmt.Printf("--setting up\n")
	switch param.Client {
	case "py":
		return setUpPython(param)
	case "ts":
		return setUpTS(param)
	default:
		panic("unrecognized client in setup")
	}
}

func setUpPython(param SetUpParam) error {
	var (
		stdErr, stdOut bytes.Buffer
		cmd            = exec.Command("sh", "-c", "poetry", "install", "&&", "poetry",
			"run", "python", "setup.py",
			strconv.Itoa(param.IndexChannels),
			strconv.Itoa(param.DataChannels),
		)
	)

	cmd.Dir = "./py"
	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut

	if err := cmd.Run(); err != nil {
		return errors.Newf(
			"err: %s\nstderr: %s\nstdout: %s",
			err.Error(),
			stdErr.String(),
			stdOut.String(),
		)
	}
	return nil
}

func setUpTS(param SetUpParam) error {
	cmd := exec.Command("sh", "-c", "npx", "tsx", "setup.ts",
		strconv.Itoa(param.IndexChannels),
		strconv.Itoa(param.DataChannels),
	)

	cmd.Dir = "./ts/src"
	var stdErr, stdOut bytes.Buffer
	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut

	if err := cmd.Run(); err != nil {
		return errors.Newf(
			"err: %s\nstderr: %s\nstdout: %s",
			err.Error(),
			stdErr.String(),
			stdOut.String(),
		)
	}
	return nil
}
