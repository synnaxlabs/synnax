import {
  DetailedHTMLProps,
  InputHTMLAttributes,
  RefAttributes,
  forwardRef,
} from "react";

import "./Input.css";
import clsx from "clsx";

import { ComponentSize } from "@/util";

export interface BaseInputProps
  extends Omit<
    DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>,
    "size" | "ref"
  > {}

export interface InputProps extends BaseInputProps, RefAttributes<HTMLInputElement> {
  size?: ComponentSize;
  name?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ size = "medium", placeholder, value, className, ...props }, ref) => {
    return (
      <input
        ref={ref}
        placeholder={placeholder}
        className={clsx("pluto-input__input", "pluto-input--" + size, className)}
        {...props}
        value={value}
      />
    );
  }
);
Input.displayName = "Input";
