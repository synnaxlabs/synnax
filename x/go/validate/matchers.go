// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate

import (
	"github.com/onsi/gomega/gcustom"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/x/errors"
)

func ContainPath(path []string) types.GomegaMatcher {
	return gcustom.MakeMatcher(func(err error) (bool, error) {
		var pathError PathError
		if !errors.As(err, &pathError) {
			return false, nil
		}
		if pathError.joinPath() == (PathError{Path: path}).joinPath() {
			return true, nil
		}
		return false, nil
	})
}
