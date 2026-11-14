// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type theme } from "@synnaxlabs/lyra/theme";
import { color } from "@synnaxlabs/x";

/**
 * @returns the color with its OKLCH lightness clamped into the theme's symbol band, the
 * same rule as the `lyra-symbol-colored` CSS class, so a user color reads on the theme.
 */
export const legible = (c: color.Color, t: theme.Theme): color.Color => {
  const { lMin, lMax, lExtreme, cThresh } = t.symbol;
  const [l, ch, h, a] = color.oklch(c);
  const gray = Math.min(1, Math.max(0, (cThresh - ch) / cThresh));
  const clamped = Math.min(lMax, Math.max(lMin, l)) * (1 - gray) + lExtreme * gray;
  return color.fromOKLCH([clamped, ch, h, a]);
};

/** @returns the color with its OKLCH lightness set, keeping its chroma and hue. */
export const withLightness = (c: color.Color, l: number, alpha = 255): color.Color => {
  const [, ch, h] = color.oklch(c);
  return color.fromOKLCH([l, ch, h, alpha / 255]);
};
