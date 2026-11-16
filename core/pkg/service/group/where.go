// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

// MatchNested returns a filter for groups that sit in a parent when nested is true, and
// for top groups when nested is false.
func MatchNested(nested bool) Filter { return MatchHasParent(nested) }
