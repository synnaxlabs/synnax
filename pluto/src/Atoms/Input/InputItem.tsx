import { forwardRef } from "react";
import { Space } from "../Space";
import { Input, InputProps } from "./Input";
import { InputHelpText } from "./InputHelpText";
import { InputLabel } from "./InputLabel";
import clsx from "clsx";

export interface InputItemProps extends InputProps {
  label?: string;
  helpText?: string;
}

export const InputItem = forwardRef<HTMLInputElement, InputItemProps>(
  ({ label, helpText, style, className, ...props }: InputItemProps, ref) => {
    return (
      <Space
        size="small"
        className={clsx("pluto-input-item", className)}
        direction="vertical"
        style={style}
      >
        <InputLabel>{label}</InputLabel>
        <Input ref={ref} {...props} />
        <InputHelpText>{helpText}</InputHelpText>
      </Space>
    );
  }
);
