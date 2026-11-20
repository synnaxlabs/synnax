// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { theme } from "@/theme";

/** A color with a name and a value for each theme mode. */
export interface Named extends theme.Pair {
  key: string;
  name: string;
}

/** Named colors that a picker shows under one title. */
export interface Palette {
  key: string;
  name: string;
  colors: Named[];
}

/** The named colors that a picker offers, and the one that its value follows. */
export interface Library {
  palettes: Palette[];
  /** The key of the named color that the value follows. Absent for a fixed color. */
  named?: string;
  /** Called with the key of the named color that the user picks. */
  onNamedChange: (key: string) => void;
}

/** @returns the named color with the key, or undefined when no palette holds it. */
export const findNamed = (palettes: Palette[], key: string): Named | undefined => {
  for (const p of palettes) {
    const found = p.colors.find((c) => c.key === key);
    if (found != null) return found;
  }
  return undefined;
};
