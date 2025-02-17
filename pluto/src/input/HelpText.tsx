// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/HelpText.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type Status } from "@/status";
import { Text } from "@/text";

/** Props for the {@link HelpText} component. */
export interface HelpTextProps extends Omit<Text.TextProps<"small">, "level" | "ref"> {
  variant?: Status.Variant;
}

/**
 * Help text for an input component. We generally recommend using Input.Item with a
 * 'helpText' prop instead of this component. This component is useful for low-level
 * control over the help text element.
 *
 * @param props - Props for the help text component. Unlisted props are passed to the
 * underlying text element.
 * @param props.variant - The variant of the help text.
 * "success" | "error" | "warning" | "info" | "loading" | "disabled
 * @default "info"
 */
export const HelpText = ({
  className,
  variant = "error",
  ...rest
}: HelpTextProps): ReactElement => (
  <Text.Text<"small">
    className={CSS(
      CSS.B("input-help-text"),
      CSS.BM("input-help-text", variant),
      className,
    )}
    level="small"
    {...rest}
  />
);
