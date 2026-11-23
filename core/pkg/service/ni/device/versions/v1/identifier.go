// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"regexp"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	"go.uber.org/zap"
)

// identifierPattern is a frozen copy of the identifier rule as of this migration.
var identifierPattern = regexp.MustCompile(`^$|^[A-Za-z][A-Za-z0-9_]{1,11}$`)

// IdentifierMigration clears each stored identifier that breaks the identifier rule,
// which earlier Cores did not check. The device is then unconfigured until the user
// gives it a valid identifier.
var IdentifierMigration = migrate.NewInstrumentedEntry(
	"v63_ni_device_identifier",
	func(_ context.Context, d Device, ins alamos.Instrumentation) (Device, error) {
		if identifierPattern.MatchString(d.Identifier) {
			return d, nil
		}
		ins.L.Warn(
			"cleared an NI device identifier that breaks the identifier rule",
			zap.Stringer("key", d.Key),
			zap.String("identifier", d.Identifier),
		)
		d.Identifier = ""
		return d, nil
	},
)
