import { ButtonHTMLAttributes, ReactElement } from "react";

import "./Button.css";
import clsx from "clsx";

import { Typography, Text } from "@/atoms/Typography";
import { ComponentSize } from "@/util/component";

/** The variant of button */
export type ButtonVariant = "filled" | "outlined" | "text";

/** The base props accepted by all button types in this directory. */
export interface BaseButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ComponentSize;
}

/** The props for the {@link Button} component. */
export interface ButtonProps extends BaseButtonProps {
  children?: string | number;
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
}

export const Button = ({
  size = "medium",
  variant = "filled",
  className,
  startIcon,
  endIcon,
  children,
  ...props
}: ButtonProps): JSX.Element => (
  <button
    className={clsx("pluto--" + size, "pluto-btn", "pluto-btn--" + variant, className)}
    {...props}
  >
    <Text.WithIcon
      color={variant === "filled" ? "var(--pluto-white)" : "var(--pluto-text-color)"}
      level={Typography.ComponentSizeLevels[size]}
      startIcon={startIcon}
      endIcon={endIcon}
    >
      {children}
    </Text.WithIcon>
  </button>
);
