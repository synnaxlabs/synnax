// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert

import (
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/flagdef"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
)

// Flag names used for creating certificates.
const (
	FlagCertsDir      = "certs-dir"
	FlagCAKey         = "ca-key"
	FlagCACert        = "ca-cert"
	FlagNodeKey       = "node-key"
	FlagNodeCert      = "node-cert"
	FlagAllowKeyReuse = "allow-key-reuse"
	FlagKeySize       = "key-size"
)

//go:embed flags.json
var flagsJSON []byte

// FlagDefs are the parsed flag definitions for the cert flag set.
var FlagDefs = flagdef.MustParse(flagsJSON)

// AddFlags adds the cert flags to the given command.
func AddFlags(cmd *cobra.Command) {
	instrumentation.AddFlags(cmd)
	flagdef.MustRegister(cmd, FlagDefs)
}
