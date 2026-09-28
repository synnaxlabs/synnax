// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color } from "@synnaxlabs/lyra/color";
import { Theming } from "@synnaxlabs/lyra/theming";
import { type optional } from "@synnaxlabs/x";
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
