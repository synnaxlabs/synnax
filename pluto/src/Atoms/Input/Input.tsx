import { DetailedHTMLProps, forwardRef, InputHTMLAttributes } from "react";
import "./Input.css";
import Space from "../Space/Space";
import { uuidShort } from "../../util/uuid";
import clsx from "clsx";

export interface InputProps
  extends Omit<
    DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>,
    "size" | "onChange" | "value"
  > {
  size?: "small" | "medium";
  name?: string;
  label?: string;
  value?: string;
  onChange?: (value: string) => void;
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      size = "medium",
      name = uuidShort(),
      label,
      placeholder,
      value,
      onChange,
      className,
      ...props
    }: InputProps,
    ref
  ) => {
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
);

export default Input;
