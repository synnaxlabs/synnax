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
