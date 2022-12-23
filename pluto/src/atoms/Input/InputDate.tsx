import { forwardRef } from "react";

import { InputProps } from "./Input";

export interface InputDateProps extends InputProps {}

export const InputDate = forwardRef<HTMLInputElement, InputDateProps>(
  ({ size = "medium", ...props }, ref) => {
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
InputDate.displayName = "InputDate";
