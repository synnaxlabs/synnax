package main

import (
	"bytes"
	"os/exec"
	"strconv"

	"github.com/synnaxlabs/x/errors"
)

func writePython(p NodeParams) error {
	if err := exec.Command("cd", "py", "&&", "poetry", "install").Run(); err != nil {
		return err
	}

	args := []string{"run", "python", "write.py"}
	args = append(
		args,
		strconv.Itoa(p.numWriters),
		strconv.Itoa(p.domains),
		strconv.Itoa(p.samplesPerDomain),
		strconv.Itoa(len(p.channelGroups)),
	)

	for _, g := range p.channelGroups {
		args = append(
			args,
			strconv.Itoa(len(g.indexChannels)),
			strconv.Itoa(len(g.dataChannels)),
		)
		args = append(args, g.indexChannels...)
		args = append(args, g.dataChannels...)
	}

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
