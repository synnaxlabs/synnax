// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { color } from "@synnaxlabs/client";
import { theme } from "@synnaxlabs/lyra/theme";
import { color as xcolor } from "@synnaxlabs/x";

/**
 * @returns the value of the color reference for the theme, or undefined for a named
 * color that named does not hold.
 * @param named - The named colors by key.
 */
export const resolve = (
  ref: color.Ref,
  named: Map<color.Key, color.Named>,
  t: theme.Theme,
): xcolor.Color | undefined => {
  if (ref.type === "fixed") return xcolor.construct(ref.value);
  const n = named.get(ref.key);
  return n == null ? undefined : theme.resolve(n, t);
};
