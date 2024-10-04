// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"github.com/synnaxlabs/synnax/pkg/service/auth/base"
	"github.com/synnaxlabs/x/errors"
)

var (
	Denied        = errors.Wrap(base.AuthError, "access denied")
	Granted error = nil
)