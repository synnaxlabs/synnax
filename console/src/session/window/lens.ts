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
 * How persistence splits a window-keyed slice across its windows. Every window-keyed
 * slice holds a windows record, so one lens serves them all, and the shape stays
 * declared here rather than assumed by the store.
 */
export const LENS: Persist.Lens = {
  keys: ({ windows }) => Object.keys(windows),
  // A window's partition stores the whole slice narrowed to that one window, so its
  // bytes still parse through the slice's own schema. A window with no entry narrows
  // to none: an undefined-valued key neither survives JSON nor parses.
  narrow: (slice, key) => ({
    ...slice,
    windows: key in slice.windows ? { [key]: slice.windows[key] } : {},
  }),
  widen: (into, from) => ({ ...from, windows: { ...into.windows, ...from.windows } }),
};
