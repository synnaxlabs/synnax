// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Color } from "@synnaxlabs/lyra/color";
import { Form } from "@synnaxlabs/lyra/form";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, type optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface ColorFieldProps extends optional.Optional<
  Color.FieldProps,
  "fallback"
> {}

/**
 * Edits an optional symbol color. The fallback defaults to the theme color a symbol
 * paints when its color is absent.
 */
export const ColorField = ({ fallback, ...rest }: ColorFieldProps): ReactElement => {
  const theme = Theming.use();
  return <Color.Field fallback={fallback ?? theme.colors.gray.l11} {...rest} />;
};

const primary = (theme: Theming.Theme): color.Color => theme.colors.primary.z;

const FILL_FALLBACKS: Partial<
  Record<schematic.ElementConfigType, (theme: Theming.Theme) => color.Color>
> = {
  button: primary,
  input: primary,
  select: primary,
  setpoint: primary,
  polygon: (theme) => theme.colors.gray.l1,
  off_page_reference: (theme) => theme.colors.gray.l11,
};

/** @returns the fill a symbol of the variant paints while its fill color is absent. */
export const fillFallback = (
  variant: schematic.ElementConfigType,
  theme: Theming.Theme,
): color.Color => FILL_FALLBACKS[variant]?.(theme) ?? color.ZERO;

export interface FillFieldProps extends Omit<ColorFieldProps, "path" | "fallback"> {}

/** Edits the fill color of the symbol in the form, with its variant's fallback. */
export const FillField = ({
  label = "Fill",
  ...rest
}: FillFieldProps): ReactElement => {
  const theme = Theming.use();
  const variant = Form.useFieldValue<schematic.ElementConfigType>("variant");
  return (
    <Color.Field
      path="fillColor"
      label={label}
      fallback={fillFallback(variant, theme)}
      {...rest}
    />
  );
};
