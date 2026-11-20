// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compare

import "strings"

// TextKey returns the key that orders s among other text: its Unicode simple lower
// case. Keys compare by code point. x/testdata/compare/text.json pins the order.
func TextKey(s string) string { return strings.ToLower(s) }
