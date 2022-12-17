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
