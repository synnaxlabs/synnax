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
import type { ReactElement } from "react";

import { useBinding } from "@/color/bind";

export interface InputProps extends Omit<
  Color.InputProps,
  "value" | "onChange" | "library"
> {
  /** The stored color reference. Absent when the theme picks the color. */
  value?: color.Ref;
  /**
   * Called with a named reference for a pick from a palette, a fixed reference for a
   * typed or custom color, and undefined for Auto.
   */
  onChange: (value?: color.Ref) => void;
}

/**
 * A swatch joined to a hex text box that picks a named color from a palette or a fixed
 * color. While the value is a named color, the box shows its name.
 */
export const Input = ({ value, onChange, ...rest }: InputProps): ReactElement => {
  const binding = useBinding(value, onChange);
  return <Color.Input {...rest} {...binding} />;
};
