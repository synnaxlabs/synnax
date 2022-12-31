// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import clsx from "clsx";

import { Input, InputProps } from "./Input";
import "./InputSwitch.css";

export const InputSwitch = forwardRef<HTMLInputElement, InputProps>(
  ({ className, size = "medium", ...props }: InputProps, ref) => {
    return (
      <div className={clsx("pluto-input-switch__container", `pluto-input--${size}`)}>
        <label className={clsx("pluto-input-switch__track", className)}>
          <Input
            className="pluto-input-switch__input"
            type="checkbox"
            ref={ref}
            {...props}
          />
          <span className="pluto-input-switch__slider"></span>
        </label>
      </div>
    );
  }
);
InputSwitch.displayName = "InputSwitch";
