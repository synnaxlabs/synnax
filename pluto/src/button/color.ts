// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Variant } from "@/button/Button";

import { type Color } from "..";

export const color = (
  variant: Variant,
  disabled?: boolean,
  color?: Color.Crude,
): Color.Crude => {
  if (disabled === true) return "var(--pluto-gray-m0)";
  if (color != null) return color;
  if (variant === "filled") return "var(--pluto-white)";
  return "var(--pluto-text-color)";
};
