package main

import (
	"bytes"
	"fmt"
	"github.com/synnaxlabs/x/errors"
	"os/exec"
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
	return nil
}

func cleanUpPython(param CleanUpParam) error {
	if !param.DeleteAllChannels {
		return nil
	}
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}
	cmd := exec.Command("poetry", "run", "python", "delete_all.py")

	cmd.Dir = "./py"
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return errors.Newf("err: %s\nstderr: %s\nstdout: %s", err.Error(), stderr.String(), stdout.String())
	}
	return nil
}
