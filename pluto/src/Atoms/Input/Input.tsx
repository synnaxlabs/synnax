import {
  DetailedHTMLProps,
  forwardRef,
  InputHTMLAttributes,
  RefAttributes,
} from "react";
import "./Input.css";
import clsx from "clsx";
import { ComponentSize } from "@/util";

interface BaseInputProps
  extends Omit<
    DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>,
    "size" | "ref"
  > {
  size?: ComponentSize;
  name?: string;
  label?: string;
}

export interface InputProps
  extends BaseInputProps,
    RefAttributes<HTMLInputElement> {}

export const Input = forwardRef<HTMLInputElement, BaseInputProps>(
  (
    { size = "medium", label, placeholder, value, className, ...props },
    ref
  ) => {
    return (
      <input
        ref={ref}
        placeholder={placeholder}
        className={clsx(
          "pluto-input__input",
          "pluto-input__input--" + size,
          className
        )}
        value={value}
        {...props}
      />
    );
  }
);
