// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build (!invariants && !tracing) || race
// +build !invariants,!tracing race

package invariants

// SetFinalizer is a wrapper around runtime.SetFinalizer that is a no-op under
// race builds or if neither the invariants or tracing build tags are
// specified.
func SetFinalizer(obj, finalizer interface{}) {
}
