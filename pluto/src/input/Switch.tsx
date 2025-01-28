// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Switch.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type BaseProps } from "@/input/types";

export interface SwitchProps extends Omit<BaseProps<boolean>, "placeholder"> {}

const CLS = "input-switch";

/**
 * A controlled boolean Switch input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @default "medium"
 */
export const Switch = ({
  ref,
  className,
  value,
  disabled,
  onChange,
  size = "medium",
  variant,
  ...props
}: SwitchProps): ReactElement => {
  if (variant === "preview") disabled = true;
  return (
    <div
      className={CSS(CSS.BE(CLS, "container"), CSS.disabled(disabled), CSS.size(size))}
    >
      <label className={CSS(CSS.BE(CLS, "track"), className)}>
        <input
          className={CSS.BE(CLS, "input")}
          type="checkbox"
          ref={ref}
          checked={value}
          onChange={(e) => onChange(e.target.checked)}
          value=""
          disabled={disabled}
          {...props}
        />
        <span className="pluto-input-switch__slider" />
      </label>
    </div>
  );
};
