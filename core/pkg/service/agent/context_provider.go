// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package agent

import "context"

// ContextProvider supplies a section of context to the LLM prompt. Each provider
// is responsible for a single category of information (channels, devices, history,
// etc.) and formats it as a human-readable string that gets appended to the user
// message.
type ContextProvider interface {
	Name() string
	BuildContext(ctx context.Context, query string) (string, error)
}
