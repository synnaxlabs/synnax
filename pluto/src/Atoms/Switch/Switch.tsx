import clsx from "clsx";
import { HTMLAttributes } from "react";
import "./Switch.css";

export interface SwitchProps extends HTMLAttributes<HTMLInputElement> {}

export const Switch = ({ className, ...props }: SwitchProps) => {
  return (
    <label className={clsx("pluto-switch__container", className)}>
      <input className="pluto-switch__input" type="checkbox" {...props} />
      <span className="pluto-switch__slider"></span>
    </label>
  );
};
