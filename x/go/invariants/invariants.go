// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package invariants is the only home of the checks that run only in test builds. A
// check inside `if invariants.Enabled { ... }` runs in a build with the invariants or
// the race tag, as the Go tests of Linux and Windows CI are, and a release build drops
// it.
package invariants
