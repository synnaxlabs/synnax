// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

// ErrDenied is returned when a subject does not have permission to perform an
// action on a resource.
var ErrDenied = auth.ErrAccessDenied

// Denied returns an error that wraps ErrDenied and names the subject, the action, and
// the object that the subject cannot act on.
func Denied(subject resource.ID, action Action, object resource.ID) error {
	return errors.Wrapf(ErrDenied, "%s cannot %s %s", subject, action, object)
}
