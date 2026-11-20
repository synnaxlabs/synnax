// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Named } from "@/color/types.gen";
import type { group } from "@/group";

/** The key of the palette that holds the series colors, which new plot lines take. */
export const SERIES_PALETTE: group.Key = "4d2d0872-30f9-4573-92da-2b72db8f8c64";

/**
 * Compares two named colors by their position in a palette, and then by key, as the
 * Core orders the series colors. A color with no position comes last.
 */
export const compare = (a: Named, b: Named): number => {
  if (a.position !== b.position) {
    if (a.position == null) return 1;
    if (b.position == null) return -1;
    return a.position < b.position ? -1 : 1;
  }
  if (a.key === b.key) return 0;
  return a.key < b.key ? -1 : 1;
};
