import { ButtonHTMLAttributes, cloneElement } from "react";
import "./Button.css";
import IconText, { BaseIconTextProps } from "../Typography/IconText";
import clsx from "clsx";
import { FontLevel } from "../../Theme/theme";

interface BaseButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "filled" | "outlined" | "text";
  size?: "small" | "medium";
}

export interface ButtonProps
  extends BaseButtonProps,
    Omit<BaseIconTextProps, "level"> {
  children: string | number;
}

const sizeLevels: { [key: string]: FontLevel } = {
  small: "small",
  medium: "p",
  large: "h4",
};

export default function Button({
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
        level={sizeLevels[size]}
        startIcon={startIcon}
        endIcon={endIcon}
      >
        {children}
      </IconText>
    </button>
  );
}

export interface IconButtonProps extends BaseButtonProps {
  children: React.ReactElement;
}

export const IconButton = ({
  children,
  className,
  variant = "text",
  size = "medium",
  ...props
}: IconButtonProps) => {
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
