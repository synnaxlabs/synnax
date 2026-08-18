// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/HelpText.css";

import { status } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Text } from "@/text";

/** Props for the {@link HelpText} component. */
export interface HelpTextProps extends Omit<
  Text.TextProps<"small">,
  "level" | "ref" | "variant"
> {
  variant?: status.Variant;
}

/**
 * Hint or error text below an input, colored by its status variant. Prefer {@link Item}
 * with a `helpText` prop; reach for this only to place the text yourself.
 */
export const HelpText = ({
  className,
  variant,
  ...rest
}: HelpTextProps): ReactElement => (
  <Text.Text<"small">
    className={CSS(CSS.B("input-help-text"), className)}
    color={9}
    status={status.removeVariants(variant, "success")}
    level="small"
    {...rest}
  />
);
