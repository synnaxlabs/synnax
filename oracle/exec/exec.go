// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package exec

import (
	"os/exec"
	"strings"

	"github.com/synnaxlabs/oracle/output"
	"github.com/synnaxlabs/x/errors"
)

func OnFiles(
	command []string,
	files []string,
	dir string,
) error {
	output.PostWriteStep(strings.Join(command, " "), len(files), "running")
	eslintArgs := append(command[1:], files...)
	eslintCmd := exec.Command(command[0], eslintArgs...)
	eslintCmd.Dir = dir
	if err := eslintCmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to run %s", command[0])
	}
	return nil
}
