import clsx from "clsx";

import { StatusVariant } from "@/atoms/Status";
import { Text, TextProps } from "@/atoms/Typography";
import "./InputHelpText.css";

export interface InputHelpTextProps extends Omit<TextProps, "level"> {
  variant?: StatusVariant;
}

export const InputHelpText = ({
  className,
  variant = "error",
  ...props
}: InputHelpTextProps): JSX.Element => (
  <Text
    className={clsx(
      "pluto-input-help-text",
      `pluto-input-help-text--${variant}`,
      className
    )}
    level="small"
    {...props}
  />
);
