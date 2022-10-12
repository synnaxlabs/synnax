import { DetailedHTMLProps, forwardRef, InputHTMLAttributes, Ref } from "react";
import "./Input.css";
import clsx from "clsx";
import { ComponentSize } from "../../util/types";

export interface InputProps
  extends Omit<
    DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>,
    "size" | "onChange" | "value"
  > {
  size?: ComponentSize;
  name?: string;
  label?: string;
  value?: string;
  onChange?: (value: string) => void;
}

function Input(
  {
    size = "medium",
    name,
    label,
    placeholder,
    value,
    onChange,
    className,
    ...props
  }: InputProps,
  ref: Ref<HTMLInputElement>
) {
  return (
    <input
      ref={ref}
      id={name}
      placeholder={placeholder}
      className={clsx(
        "pluto-input__input",
        "pluto-input__input--" + size,
        className
      )}
      onChange={(e) => {
        if (onChange) onChange(e.target.value);
      }}
      value={value}
      {...props}
    />
  );
}

export default forwardRef(Input);
