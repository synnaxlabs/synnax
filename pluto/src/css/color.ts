// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@/color/core";

/**
 * Generates a set of CSS variables representing different opacities of a given
 * color.
 * @param prefix The prefix to use for the CSS variable names.
 * @param hex The color to generate opacities for.
 * @param opacities A list of the opacities to generate
 * @returns Record mapping the CSS variable names to their values.
 */
export const createHexOpacityVariants = (
  prefix: string,
  hex: color.Crude,
  opacities: readonly number[],
): Record<string, string> =>
  Object.fromEntries(
    opacities.map((o) => [`${prefix}-${o}`, new color.Color(hex).setAlpha(o).hex]),
  );
