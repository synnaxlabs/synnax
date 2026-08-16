// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import "github.com/synnaxlabs/x/gorp"

var _ gorp.Entry[string] = SecureCredentials{}

// GorpKey implements gorp.Entry.
func (sc SecureCredentials) GorpKey() string { return sc.Username }

// SetOptions implements gorp.Entry.
func (SecureCredentials) SetOptions() []any { return nil }
