import clsx from "clsx";
import { cloneElement } from "react";
import { BaseButtonProps } from "./Button";

export interface ButtonIconOnlyProps extends BaseButtonProps {
  /** The icon to render */
  children: React.ReactElement;
}

const ButtonIconOnly = ({
  children,
  className,
  variant = "text",
  size = "medium",
  ...props
}: ButtonIconOnlyProps) => {
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

export default ButtonIconOnly;
