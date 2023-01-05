// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import clsx from "clsx";

import { InputBaseProps } from "./types";

import "./InputSwitch.css";

export interface InputSwitchProps extends InputBaseProps<boolean> {}

export const InputSwitch = forwardRef<HTMLInputElement, InputSwitchProps>(
  (
    { className, value, onChange, size = "medium", ...props }: InputSwitchProps,
    ref
  ) => (
    <div className={clsx("pluto-input-switch__container", `pluto--${size}`)}>
      <label className={clsx("pluto-input-switch__track", className)}>
        <input
          className="pluto-input-switch__input"
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
