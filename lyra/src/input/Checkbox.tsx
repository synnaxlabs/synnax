// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Checkbox.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Boolean, type BooleanProps } from "@/input/Boolean";

/** Props for {@link Checkbox}. */
export interface CheckboxProps extends Omit<BooleanProps, "inputType"> {
  /**
   * Draws a dash in place of the check, for a choice that is partly made, such as a
   * folder with some of its items picked.
   */
  indeterminate?: boolean;
}

/** A boolean input drawn as a checkbox. Use it for a choice the user then submits. */
export const Checkbox = ({
  indeterminate = false,
  className,
  ...rest
}: CheckboxProps): ReactElement => (
  <Boolean
    {...rest}
    inputType="checkbox"
    className={CSS.cls(indeterminate && CSS.M("indeterminate"), className)}
  />
);
