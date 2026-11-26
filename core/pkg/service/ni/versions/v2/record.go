// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

// AIBase returns the fields that every analog input variant shares. Each variant with
// shared fields embeds BaseAIChannel, so each one has AIBase.
func (b BaseAIChannel) AIBase() BaseAIChannel { return b }
