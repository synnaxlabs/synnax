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
	"fmt"

	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// ErrDenied is returned when a subject does not have permission to perform an
// action on a resource.
var ErrDenied = auth.ErrAccessDenied

// deniedError is the error of a subject that cannot act on an object. It wraps
// ErrDenied.
type deniedError struct {
	// Subject is the subject that asked.
	Subject resource.ID
	// Action is what the subject asked to do.
	Action Action
	// Object is the record that the subject cannot act on.
	Object resource.ID
}

// Error implements error.
func (e deniedError) Error() string {
	return fmt.Sprintf("%s cannot %s %s: %s", e.Subject, e.Action, e.Object, ErrDenied)
}

// Unwrap returns ErrDenied.
func (deniedError) Unwrap() error { return ErrDenied }

// Denied returns an error that wraps ErrDenied and names the subject, the action, and
// the object that the subject cannot act on.
func Denied(subject resource.ID, action Action, object resource.ID) error {
	return deniedError{Subject: subject, Action: action, Object: object}
}
