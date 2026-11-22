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
import { color as xcolor } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { useBinding } from "@/color/bind";

interface BaseProps extends Omit<
  Color.SwatchProps,
  "value" | "onChange" | "fallback" | "library"
> {}

interface RequiredValueProps extends BaseProps {
  /** The stored color reference. */
  value: color.Ref;
  /**
   * Called with a named reference for a pick from a palette and a fixed reference for a
   * custom color. Without it, the swatch only shows the color.
   */
  onChange?: (value: color.Ref) => void;
  fallback?: undefined;
}

interface OptionalValueProps extends BaseProps {
  /** The stored color reference. Absent when the theme picks the color. */
  value?: color.Ref;
  /**
   * Called with a named reference for a pick from a palette, a fixed reference for a
   * custom color, and undefined for Auto. Without it, the swatch only shows the color.
   */
  onChange?: (value?: color.Ref) => void;
  /**
   * The color that the theme paints while the value is absent. Setting it makes the
   * value optional and offers Auto.
   */
  fallback: xcolor.Crude;
}

export type SwatchProps = RequiredValueProps | OptionalValueProps;

/** A swatch that picks a named color from a palette or a fixed color. */
export const Swatch = (props: SwatchProps): ReactElement => {
  const { value, onChange, fallback: _, ...rest } = props;
  const binding = useBinding(value, (ref) => {
    if (ref != null) onChange?.(ref);
    else if (props.fallback != null) props.onChange?.(undefined);
  });
  const change = onChange == null ? undefined : binding.onChange;
  if (props.fallback == null)
    return (
      <Color.Swatch
        {...rest}
        value={binding.value ?? xcolor.TRANSPARENT}
        onChange={change}
        library={binding.library}
      />
    );
  return (
    <Color.Swatch
      {...rest}
      value={binding.value}
      fallback={props.fallback}
      onChange={change}
      library={binding.library}
    />
  );
};
