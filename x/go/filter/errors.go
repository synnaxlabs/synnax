// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package filter

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ErrNoOperator is the error for an operator that sets no member that selects values,
// such as `name: {}`. Such an operator would match every value.
var ErrNoOperator = errors.Wrap(validate.ErrValidation, "no operator set")
