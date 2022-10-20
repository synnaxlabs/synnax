import clsx from "clsx";
import { Text, TextProps } from "../Typography";
import "./InputHelpText.css";

type HelpTextVariant = "error" | "info" | "success" | "warning";

export interface InputHelpTextProps extends Partial<TextProps> {
  variant?: HelpTextVariant;
}

export default function InputHelpText({
  level,
  variant = "error",
  ...props
}: InputHelpTextProps) {
  return (
    <Text
      className={clsx(
        "pluto-input-help-text",
        `pluto-input-help-text--${variant}`,
        props.className
      )}
      level="small"
      {...props}
    />
  );
}
