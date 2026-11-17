// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// NewErrorCodec returns the codec for the errors that the Core and its clients send
// to each other.
func NewErrorCodec() *errors.Codec {
	return errors.NewCodec(
		freighter.ErrorProvider(),
		query.ErrorProvider(),
		validate.ErrorProvider(),
		control.ErrorProvider(),
		arc.ErrorProvider(),
		auth.ErrorProvider(),
		license.ErrorProvider(),
	)
}
