import { useState } from "react";

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { Input } from "@/input";

import { type BaseProps } from "./types";

export interface DateTimeProps extends BaseProps<number> {}

export const DateTime = ({ value, onChange, onBlur, ...props }: DateTimeProps) => {
  const ts = new TimeStamp(value, "UTC");
  const [internalValue, setInternalVlaue] = useState(
    ts.fString("ISO", "local").slice(0, -1),
  );
  const [valueIsValid, setValueIsValid] = useState(true);

  const handleChange = (next: string | number): void => {
    setInternalVlaue(next.toString());
    let ts = new TimeStamp(next, "UTC");
    ts = ts.add(
      (TimeStamp.now().date().getTimezoneOffset() - ts.date().getTimezoneOffset()) *
        TimeSpan.MINUTE.valueOf(),
    );
    let ok = false;
    try {
      const str = ts.fString("ISO", "local");
      ok = str.slice(0, -1) === next.toString();
    } catch (_) {}
    if (!ok) {
      setValueIsValid(false);
      return;
    }
    onChange(ts.valueOf());
    setValueIsValid(true);
  };

  const handleBlur: React.FocusEventHandler<HTMLInputElement> = (e) => {
    setValueIsValid(true);
    setInternalVlaue(new TimeStamp(value, "UTC").fString("ISO", "local").slice(0, -1));
    onBlur?.(e);
  };

  return (
    <Input.Text
      type="datetime-local"
      onBlur={handleBlur}
      required={false}
      value={
        valueIsValid
          ? new TimeStamp(value, "UTC").fString("ISO", "local").slice(0, -1)
          : internalValue
      }
      onChange={handleChange}
      {...props}
    />
  );
};
