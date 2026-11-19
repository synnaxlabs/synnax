// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/client";
import { Color } from "@synnaxlabs/lyra/color";
import { type color as xcolor } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { usePalettes, useResolve } from "@/color/library";

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
export const Swatch = ({
  value,
  onChange,
  fallback,
  ...rest
}: SwatchProps): ReactElement => {
  const palettes = usePalettes();
  const resolve = useResolve();
  return (
    <Color.Swatch
      value={value == null ? undefined : resolve(value)}
      fallback={fallback}
      onChange={(c) => onChange(c == null ? undefined : { type: "fixed", value: c })}
      library={
        palettes.length === 0
          ? undefined
          : {
              palettes,
              named: value?.type === "named" ? value.key : undefined,
              onNamedChange: (key) => onChange({ type: "named", key }),
            }
      }
      {...rest}
    />
  );
};
