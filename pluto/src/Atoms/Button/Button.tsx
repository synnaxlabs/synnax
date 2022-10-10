import { ButtonHTMLAttributes, cloneElement, ReactElement } from "react";
import "./Button.css";
import { ComponentSizeTypographyLevels, IconText } from "../Typography";
import clsx from "clsx";
import { ComponentSize } from "../../util/types";

interface BaseButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "filled" | "outlined" | "text";
  size?: ComponentSize;
}

export interface ButtonProps extends BaseButtonProps {
  children: string | number;
  startIcon?: ReactElement;
  endIcon?: ReactElement;
}

function Button({
  size = "medium",
  variant = "filled",
  className,
  startIcon,
  endIcon,
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      className={clsx(
        "pluto-btn",
        "pluto-btn--" + variant,
        "pluto-btn--" + size,
        className
      )}
      {...props}
    >
      <IconText
        level={ComponentSizeTypographyLevels[size]}
        startIcon={startIcon}
        endIcon={endIcon}
      >
        {children}
      </IconText>
    </button>
  );
}

export interface ButtonIconOnlyProps extends BaseButtonProps {
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

Button.IconOnly = ButtonIconOnly;

export default Button;
