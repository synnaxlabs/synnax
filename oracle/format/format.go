// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package format provides language-aware formatters that produce canonical
// output for oracle's generated code. Plugins emit raw template output; the
// format layer turns that into the byte-identical state that would otherwise
// be produced by running gofmt / prettier / clang-format / etc. on the file
// after writing it to disk. That lets the sync step compare in-memory
// formatter output against the existing on-disk file and skip the write when
// they match.
package format

import "context"

// Formatter produces canonical bytes for a single generated file.
//
// Implementations must be idempotent: Format(Format(x)) == Format(x). They
// must also be deterministic given the same content and path; the sync
// layer relies on byte-identical output across invocations to decide
// whether a write is needed. ctx carries cancellation from the parent
// sync call so long-running shell-outs (eslint, prettier) terminate when
// the caller aborts.
type Formatter interface {
	Format(ctx context.Context, content []byte, absPath string) ([]byte, error)
}
