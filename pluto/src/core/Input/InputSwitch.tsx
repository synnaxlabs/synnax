// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import { InputBaseProps } from "./types";

import "@/core/Input/InputSwitch.css";
import { CSS } from "@/css";

export interface InputSwitchProps extends InputBaseProps<boolean> {}

const CLS = "input-switch";

export const InputSwitch = forwardRef<HTMLInputElement, InputSwitchProps>(
  (
    { className, value, onChange, size = "medium", ...props }: InputSwitchProps,
    ref
  ) => (
    <div className={CSS(CSS.BE(CLS, "container"), CSS.size(size))}>
      <label className={CSS(CSS.BE(CLS, "track"), className)}>
        <input
          className={CSS.BE(CLS, "input")}
          type="checkbox"
          ref={ref}
          checked={value}
          onChange={(e) => onChange(e.target.checked)}
          value=""
          {...props}
        />
        <span className="pluto-input-switch__slider" />
      </label>
    </div>
  )
);
InputSwitch.displayName = "InputSwitch";
