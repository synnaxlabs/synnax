// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package git

import (
	"os/exec"
	"strings"
)

func CurrentCommit(short ...bool) (string, error) {
	args := []string{"rev-parse"}
	if len(short) > 0 && short[0] {
		args = append(args, "--short")
	}
	args = append(args, "HEAD")
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.Trim(string(out), "\n"), nil
}
