// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color } from "@synnaxlabs/lyra/color";
import { type theme } from "@synnaxlabs/lyra/theme";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type ColorFallback } from "@/schematic/node/spec";

/** @returns the theme color a symbol paints while one of its colors is absent. */
export const defaultFallback = (theme: theme.Theme): color.Color =>
  theme.colors.gray.l11;

/** @returns the fill interactive symbols paint while their fill color is absent. */
export const primaryFallback = (theme: theme.Theme): color.Color =>
  theme.colors.primary.z;

export interface ColorFieldProps extends Omit<Color.FieldProps, "fallback"> {
  /** The color the symbol paints while the field is absent. */
  fallback?: ColorFallback;
}

/** Edits an optional symbol color. */
export const ColorField = ({
  fallback = defaultFallback,
  ...rest
}: ColorFieldProps): ReactElement => {
  const theme = Theming.use();
  return <Color.Field fallback={fallback(theme)} {...rest} />;
};

/** @returns no fill, the fill most symbols paint while their fill color is absent. */
export const noFillFallback: ColorFallback = () => color.ZERO;

export interface FillFieldProps extends Omit<ColorFieldProps, "path"> {}

/** Edits the fill color of the symbol in the form. The fallback defaults to no fill. */
export const FillField = ({
  label = "Fill",
  fallback = noFillFallback,
  ...rest
}: FillFieldProps): ReactElement => (
  <ColorField path="fillColor" label={label} fallback={fallback} {...rest} />
);
