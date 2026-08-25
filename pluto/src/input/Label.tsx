// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Label.css";

import { type DetailedHTMLProps, type HTMLAttributes, type ReactElement } from "react";

import { CSS } from "@/css";

/** Props for the {@link Label} component. */
export interface LabelProps extends DetailedHTMLProps<
  HTMLAttributes<HTMLLabelElement>,
  HTMLLabelElement
> {
  required?: boolean;
}

/**
 * A styled `label` element, marking a required field with an asterisk. Prefer
 * {@link Item} with a `label` prop; reach for this only to place the label yourself.
 */
export const Label = ({
  className,
  required = false,
  children,
  ...rest
}: LabelProps): ReactElement => (
  <label className={CSS.cls(CSS.B("input-label"), className)} {...rest}>
    {children} {required && <span className={CSS.B("required-indicator")}>*</span>}
  </label>
);
