import { cloneElement, ReactElement } from "react";

import clsx from "clsx";

import { BaseButtonProps } from "./Button";

/** The props for the {@link ButtonIconOnly} */
export interface ButtonIconOnlyProps extends BaseButtonProps {
  children: ReactElement;
}

export const ButtonIconOnly = ({
  children,
  className,
  variant = "text",
  size = "medium",
  ...props
}: ButtonIconOnlyProps): JSX.Element => (
  <button
    className={clsx(
      "pluto-btn pluto-btn-icon",
      "pluto--" + size,
      "pluto-btn--" + variant,
      className
    )}
    {...props}
  >
    {cloneElement(children, { className: "pluto-btn-icon__icon" })}
  </button>
);
