// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package os

import (
	"os"

	"github.com/synnaxlabs/x/errors"
)

func WriteTemp(dir string, prefix string, data []byte) (string, error) {
	cfgFile, err := os.CreateTemp(dir, prefix)
	if err != nil {
		return cfgFile.Name(), err
	}
	defer func() { err = errors.Combine(err, cfgFile.Close()) }()
	_, err = cfgFile.Write(data)
	return cfgFile.Name(), err
}
