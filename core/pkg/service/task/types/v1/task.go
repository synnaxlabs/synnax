// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"fmt"

	rackv1 "github.com/synnaxlabs/synnax/pkg/service/rack/types/v1"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Task{}

// GorpKey implements gorp.Entry.
func (t Task) GorpKey() Key { return t.Key }

// SetOptions implements gorp.Entry, leasing the task to its rack's node.
func (t Task) SetOptions() []any { return []any{t.Key.Rack().Node()} }

// Rack returns the key of the rack the task belongs to.
func (t Task) Rack() rackv1.Key { return t.Key.Rack() }

// String returns the task's name and key, or just the key when unnamed.
func (t Task) String() string {
	if t.Name != "" {
		return fmt.Sprintf("[%s]<%s>", t.Name, t.Key)
	}
	return t.Key.String()
}

// String returns the command's type together with its key and target task.
func (c Command) String() string {
	return fmt.Sprintf("%s (key=%s, task=%s)", c.Type, c.Key, c.Task)
}
