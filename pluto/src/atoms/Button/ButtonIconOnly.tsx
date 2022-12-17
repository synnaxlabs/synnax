import { cloneElement } from "react";

import clsx from "clsx";

import { BaseButtonProps } from "./Button";

export interface ButtonIconOnlyProps extends BaseButtonProps {
  /** The icon to render */
  children: React.ReactElement;
}

export const ButtonIconOnly = ({
  children,
  className,
  variant = "text",
  size = "medium",
  ...props
}: ButtonIconOnlyProps): JSX.Element => {
  return (
    <button
      className={clsx(
        "pluto-btn pluto-btn-icon",
        "pluto-btn--" + size,
        "pluto-btn--" + variant,
        className
      )}
      {...props}
    >
      {cloneElement(children, { className: "pluto-btn-icon__icon" })}
    </button>
  );
};
