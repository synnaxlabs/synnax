import clsx from "clsx";

import { Button, ButtonProps } from "./Button";
import { ButtonIconOnly, ButtonIconOnlyProps } from "./ButtonIconOnly";

import "./ButtonToggle.css";

export interface ButtonToggleProps extends Omit<ButtonProps, "variant"> {
  checked: boolean;
}

export const ButtonToggle = ({
  checked,
  className,
  ...props
}: ButtonToggleProps): JSX.Element => {
  return (
    <Button
      className={clsx(
        "pluto-btn-toggle",
        checked && "pluto-btn-toggle--checked",
        className
      )}
      variant={checked ? "filled" : "outlined"}
      {...props}
    />
  );
};

export interface ButtonIconToggleProps extends Omit<ButtonIconOnlyProps, "variant"> {
  checked: boolean;
}

export const ButtonToggleIcon = ({
  checked,
  className,
  ...props
}: ButtonIconToggleProps): JSX.Element => {
  return (
    <ButtonIconOnly
      variant={checked ? "filled" : "outlined"}
      className={clsx(
        "pluto-btn-toggle",
        checked && "pluto-btn-toggle--checked",
        className
      )}
      {...props}
    />
  );
};
