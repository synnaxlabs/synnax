// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import "github.com/synnaxlabs/arc/parser"

type options struct {
	// config carries parse settings used when re-parsing embedded expressions.
	config parser.Config
}

// Option configures the compiler.
type Option func(o *options)

// WithConfig sets the parse settings used when the compiler re-parses embedded
// expressions (e.g. format-string placeholders).
func WithConfig(cfg parser.Config) Option {
	return func(o *options) { o.config = cfg }
}
