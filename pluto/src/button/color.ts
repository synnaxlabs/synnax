// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color as core } from "@synnaxlabs/x";

import { type Variant } from "@/button/Button";
import { type Text } from "@/text";

export const color = (
  variant: Variant,
  disabled?: boolean,
  color?: core.Crude,
  shade?: Text.Shade,
): string | undefined => {
  if (disabled === true) return "var(--pluto-gray-l7)";
  if (color != null) return core.cssString(color);
  if (variant === "filled") return "var(--pluto-text-on-primary)";
  if (shade != null) return undefined;
  return "var(--pluto-text-color)";
};
