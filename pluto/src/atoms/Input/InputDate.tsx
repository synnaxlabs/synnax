import { forwardRef } from "react";
import { ComponentSize } from "@/util";
import { InputProps } from "./Input";

export interface InputDateProps extends InputProps {}

export const InputDate = forwardRef<HTMLInputElement, InputDateProps>(
  ({ size, ...props }, ref) => {
    return (
      <input
        ref={ref}
        type="date"
        className={`pluto-input__input pluto-input--${size}`}
        {...props}
      />
    );
  }
);
