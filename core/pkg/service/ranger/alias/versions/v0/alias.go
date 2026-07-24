// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"fmt"

	"github.com/synnaxlabs/x/gorp"
)

const keySeparator = "---"

var _ gorp.Entry[string] = Alias{}

// GorpKey implements gorp.Entry.
func (a Alias) GorpKey() string {
	return fmt.Sprintf("%s%s%s", a.Range, keySeparator, a.Channel)
}

// SetOptions implements gorp.Entry.
func (Alias) SetOptions() []any { return nil }
