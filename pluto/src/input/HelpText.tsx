// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/HelpText.css";

import { status } from "@synnaxlabs/x";
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
 * Help text for an input component. We generally recommend using Input.Item with a
 * 'helpText' prop instead of this component. This component is useful for low-level
 * control over the help text element.
 *
 * @param props - Props for the help text component. Unlisted props are passed to the
 * underlying text element.
 * @param props.variant - The variant of the help text.
 * @see {@link status.Variant}
 * @defaultValue `"error"`
 */
export const HelpText = ({
  className,
  variant,
  ...rest
}: HelpTextProps): ReactElement => (
  <Text.Text<"small">
    className={CSS(CSS.B("input-help-text"), className)}
    color={8}
    status={status.removeVariants(variant, "success")}
    level="small"
    {...rest}
  />
);
