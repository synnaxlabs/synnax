// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen pre-Oracle storage shape of stored credentials. It exists
// solely to drive the migration that lifts stored rows into the current Orc format.
package v0

import "github.com/synnaxlabs/x/gorp"

// SecureCredentials is the legacy storage shape of a credential row. Stored rows are
// MessagePack maps keyed by Go field names, so the struct must stay untagged. The type
// name must stay "SecureCredentials" so it shares the Gorp key prefix with the current
// auth.SecureCredentials.
type SecureCredentials struct {
	// Username is the unique login name the credentials belong to.
	Username string
	// Password is the bcrypt hash of the user's password.
	Password []byte
}

var _ gorp.Entry[string] = SecureCredentials{}

// GorpKey implements gorp.Entry.
func (sc SecureCredentials) GorpKey() string { return sc.Username }

// SetOptions implements gorp.Entry.
func (SecureCredentials) SetOptions() []any { return nil }
