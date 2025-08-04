// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/TextArea.css";

import { type ComponentPropsWithRef, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { type Control, type Variant } from "@/input/types";
import { type Text } from "@/text";

type HTMlTextAreaProps = Omit<
  ComponentPropsWithRef<"textarea">,
  "size" | "onChange" | "value" | "children"
>;

export interface TextAreaProps
  extends Omit<HTMlTextAreaProps, "wrap">,
    Control<string> {
  selectOnFocus?: boolean;
  centerPlaceholder?: boolean;
  variant?: Variant;
  sharp?: boolean;
  children?: ReactNode;
  wrap?: boolean;
  level?: Text.Level;
}

/**
 * A controlled text area input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @param props.selectOnFocus - Whether the input should select its contents when focused.
 * @param props.centerPlaceholder - Whether the placeholder should be centered.
 */
export const TextArea = ({
  ref,
  value,
  style,
  onChange,
  className,
  onFocus,
  selectOnFocus = false,
  variant = "outlined",
  sharp = false,
  level = "h3",
  children,
  wrap,
  ...rest
}: TextAreaProps): ReactElement => (
  <textarea
    style={style}
    className={CSS(
      CSS.B("textarea"),
      CSS.BM("textarea", variant),
      CSS.BM("text", level),
      CSS.sharp(sharp),
      className,
    )}
    ref={ref}
    value={value}
    onChange={(e) => onChange(e.target.value.replace(/\n/g, ""))}
    onFocus={(e) => {
      if (selectOnFocus) e.target.select();
      onFocus?.(e);
    }}
    {...rest}
  />
);
