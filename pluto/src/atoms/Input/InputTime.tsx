import { forwardRef } from "react";

import { Input } from "./Input";
import { InputBaseProps } from "./types";

import { nanoTimeString, timeStringNano } from "@/util/time";

export interface InputTimeProps extends InputBaseProps<number> {}

export const InputTime = forwardRef<HTMLInputElement, InputTimeProps>(
  ({ size = "medium", value, onChange, ...props }: InputTimeProps, ref) => {
    return (
      <Input
        ref={ref}
        type="time"
        step="1"
        value={nanoTimeString(value)}
        onChange={(value) => value.length > 0 && onChange(timeStringNano(value))}
        {...props}
      />
    );
  }
);
InputTime.displayName = "InputTime";
