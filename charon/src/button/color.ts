// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { type Variant } from "@/button/Button";

export const parseColor = (
  variant: Variant,
  disabled?: boolean,
  crudeColor?: color.Crude,
): string | undefined => {
  if (disabled === true) return "var(--pluto-gray-l7)";
  if (crudeColor != null) return color.cssString(crudeColor);
  if (variant === "filled") return "var(--pluto-text-on-primary)";
  return undefined;
};
