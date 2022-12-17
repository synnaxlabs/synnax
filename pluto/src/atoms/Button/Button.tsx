import { ButtonHTMLAttributes, ReactElement } from "react";

import "./Button.css";
import clsx from "clsx";

import { ComponentSizeTypographyLevels, Text } from "@/atoms/Typography";
import { ComponentSize } from "@/util";

/** The base props accepted by all button types in this directory. */
export interface BaseButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "filled" | "outlined" | "text";
  size?: ComponentSize;
}

/** The props for the {@link Button} component. */
export interface ButtonProps extends BaseButtonProps {
  children: string | number;
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
}

/**
 * Button is a basic button component.
 *
 * @param props - Props for the component, which are passed down to the underlying button
 * element.
 * @param props.size - The size of button render.
 * @param props.variant - The variant to render for the button. Options are "filled"
 * (default), "outlined", and "text".
 * @param props.startIcon - An optional icon to render before the start of the button
 * text. This can be a single icon or an array of icons. The icons will be formatted
 * to match the color and size of the button.
 * @param props.endIcon - The same as {@link startIcon}, but renders after the button
 * text.
 */
export const Button = ({
  size = "medium",
  variant = "filled",
  className,
  startIcon,
  endIcon,
  children,
  ...props
}: ButtonProps): JSX.Element => {
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
      <Text.WithIcon
        color={variant === "filled" ? "var(--pluto-white)" : "var(--pluto-text-color)"}
        level={ComponentSizeTypographyLevels[size]}
        startIcon={startIcon}
        endIcon={endIcon}
      >
        {children}
      </Text.WithIcon>
    </button>
  );
};
