package main

import (
	"bytes"
	"os/exec"
	"strconv"

	// "strconv"

	"github.com/synnaxlabs/x/errors"
)

type SetUpParam struct {
	indexChannels int
	datachannels  int
	client        string
}

func setUp(param SetUpParam) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}
	cmd := exec.Command("poetry", "run", "python", "setup.py",
		strconv.Itoa(param.indexChannels),
		strconv.Itoa(param.datachannels),
	)

	cmd.Dir = "./py"
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return errors.Newf("err: %s\nstderr: %s\nstdout: %s", err.Error(), stderr.String(), stdout.String())
	}
	return nil
}
