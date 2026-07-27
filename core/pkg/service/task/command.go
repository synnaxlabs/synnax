// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import "fmt"

// String returns the command's type together with its key and target task.
func (c Command) String() string {
	return fmt.Sprintf("%s (key=%s, task=%s)", c.Type, c.Key, c.Task)
}
