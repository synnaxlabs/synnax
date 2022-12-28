import { forwardRef } from "react";

import { Input } from "./Input";
import { InputBaseProps } from "./types";

import { isoStringShortDate, shortDateISOString } from "@/util/time";

export interface InputDateProps extends InputBaseProps<number> {}

export const InputDate = forwardRef<HTMLInputElement, InputDateProps>(
  ({ size = "medium", onChange, value, ...props }, ref) => (
    <Input
      ref={ref}
      value={shortDateISOString(value)}
      onChange={(v) => onChange(isoStringShortDate(v))}
      type="date"
      {...props}
    />
  )
);
InputDate.displayName = "InputDate";
