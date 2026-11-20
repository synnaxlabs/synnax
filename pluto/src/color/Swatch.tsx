// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { color } from "@synnaxlabs/client";
import { Color } from "@synnaxlabs/lyra/color";
import type { color as xcolor } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { useBinding } from "@/color/bind";

export interface SwatchProps extends Omit<
  Color.SwatchProps,
  "value" | "onChange" | "fallback" | "library"
> {
  /** The stored color reference. Absent when the theme picks the color. */
  value?: color.Ref;
  /**
   * Called with a named reference for a pick from a palette, a fixed reference for a
   * custom color, and undefined for Auto.
   */
  onChange: (value?: color.Ref) => void;
  /** The color that the theme paints while the value is absent. */
  fallback: xcolor.Crude;
}

/** A swatch that picks a named color from a palette or a fixed color. */
export const Swatch = ({ value, onChange, ...rest }: SwatchProps): ReactElement => {
  const binding = useBinding(value, onChange);
  return <Color.Swatch {...binding} {...rest} />;
};
