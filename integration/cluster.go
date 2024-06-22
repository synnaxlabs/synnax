package main

import (
	"fmt"
	"github.com/synnaxlabs/x/telem"
	"os"
	"os/exec"
	"time"
)

type ClusterParam struct {
	Insecure bool `json:"insecure"`
	MemFS    bool `json:"mem_fs"`
}

func startCluster(p ClusterParam) func() error {
	if p == (ClusterParam{}) {
		fmt.Printf("--cannot find cluster startup configration, skipping\n")
		return func() error { return nil }
	}

	fmt.Printf("--starting cluster\n")
	args := []string{"run", "main.go", "start", "-v"}
	if p.Insecure {
		args = append(args, "-i")
	}
	if p.MemFS {
		args = append(args, "-m")
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = "./../synnax"

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	time.Sleep(5 * telem.Second.Duration())
	return func() error { return cmd.Process.Signal(os.Interrupt) }
}
