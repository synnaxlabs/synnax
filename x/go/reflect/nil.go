// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package reflect

import "reflect"

// IsNil checks if an interface or the value it points to is nil.
func IsNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}
