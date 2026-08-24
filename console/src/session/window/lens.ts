// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Persist } from "@/session/persist";

/**
 * How persistence splits a window-keyed slice: the windows record splits one entry per
 * window partition, and every other top-level field is shared, stored once in the
 * project partition. Every window-keyed slice holds a windows record, so one lens
 * serves them all, and the shape stays declared here rather than assumed by the store.
 */
export const LENS: Persist.Lens = {
  // A window's partition stores only its own windows entry. Every other field carries
  // a schema default, so the bytes still parse through the slice's own schema. A
  // window with no entry narrows to none: an undefined-valued key neither survives
  // JSON nor parses.
  narrow: ({ windows }, key) => ({
    windows: key in windows ? { [key]: windows[key] } : {},
  }),
  shared: (slice) => ({ ...slice, windows: {} }),
  // Shared fields flow from the into side, which read them from the project
  // partition; the defaults a window partition parsed must not clobber them.
  widen: (into, from) => ({ ...into, windows: { ...into.windows, ...from.windows } }),
};
