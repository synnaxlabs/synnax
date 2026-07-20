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

	channelv0 "github.com/synnaxlabs/synnax/pkg/service/channel/types/v0"
	rangerv1 "github.com/synnaxlabs/synnax/pkg/service/ranger/types/v1"
	"github.com/synnaxlabs/x/gorp"
)

const keySeparator = "---"

// gorpKey composes the storage key for the alias of ch on range r. Kept
// package-local so the frozen key format never depends on the package root.
func gorpKey(r rangerv1.Key, ch channelv0.Key) string {
	return fmt.Sprintf("%s%s%s", r, keySeparator, ch)
}

var _ gorp.Entry[string] = Alias{}

// GorpKey implements gorp.Entry.
func (a Alias) GorpKey() string { return gorpKey(a.Range, a.Channel) }

// SetOptions implements gorp.Entry.
func (a Alias) SetOptions() []any { return nil }
