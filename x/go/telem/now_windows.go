// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "golang.org/x/sys/windows"

// Now returns the current wall clock time. Go's time.Now on Windows moves only on each
// system tick, so Now reads GetSystemTimePreciseAsFileTime instead.
func Now() TimeStamp {
	var ft windows.Filetime
	windows.GetSystemTimePreciseAsFileTime(&ft)
	return TimeStamp(ft.Nanoseconds())
}
