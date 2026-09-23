// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ErrRootCredentialsManaged is returned when a caller tries to change the root user's
// username or password. The Core reconciles both against its configuration on every
// startup, so the change would hold until the next restart and then silently revert.
var ErrRootCredentialsManaged = errors.Wrap(
	validate.ErrValidation,
	"the root user's username and password are set by the Core's configuration",
)
