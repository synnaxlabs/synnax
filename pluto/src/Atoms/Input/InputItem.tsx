import { forwardRef } from "react";
import { Space } from "../Space";
import Input, { InputProps } from "./Input";
import InputHelpText from "./InputHelpText";
import InputLabel from "./InputLabel";

export interface InputItemProps extends InputProps {
  label?: string;
  helpText?: string;
}

const InputItem = forwardRef<HTMLInputElement, InputItemProps>(
  ({ label, helpText, style, ...props }: InputItemProps, ref) => {
    return (
      <Space
        className="pluto-input-item"
        direction="vertical"
        size="small"
        style={style}
      >
        <InputLabel>{label}</InputLabel>
        <Input ref={ref} {...props} />
        <InputHelpText>{helpText}</InputHelpText>
      </Space>
    );
  }
);

export default InputItem;
