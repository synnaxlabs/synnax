// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package version provides utilities for printing the version of Synnax.
package version

import (
	"fmt"
	"io"

	"github.com/synnaxlabs/synnax/pkg/version"
)

// FPrint prints the version of Synnax to the given writer.
func FPrint(w io.Writer) error {
	_, err := fmt.Fprintf(w, "Synnax %s\n", version.Full())
	return err
}
