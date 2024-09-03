// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Text } from "@/input/Text";
import { type BaseProps } from "@/input/types";

export interface DateTimeProps extends BaseProps<number> {
  onlyChangeOnBlur?: boolean;
}

export const DateTime = ({
  value,
  onChange,
  onBlur,
  onlyChangeOnBlur,
  ...props
}: DateTimeProps): ReactElement => {
  const [tempValue, setTempValue] = useState<string | null>(null);

  const handleChange = (next: string | number, override: boolean = false): void => {
    let nextStr = next.toString();
    setTempValue(nextStr);

    let nextTS = new TimeStamp(next, "UTC");
    if (nextStr.length < 23) nextStr += ".000";

    nextTS = nextTS.add(
      BigInt(
        TimeStamp.now().date().getTimezoneOffset() - nextTS.date().getTimezoneOffset(),
      ) * TimeSpan.MINUTE.valueOf(),
    );
    let ok = false;
    try {
      const str = nextTS.fString("ISO", "local");
      ok = str.slice(0, -1) === nextStr;
    } catch (e) {
      console.error(e);
    }
    if (ok && !onlyChangeOnBlur) {
      onChange(Number(nextTS.valueOf()));
      setTempValue(null);
    }
    if (override) {
      if (ok) onChange(Number(nextTS.valueOf()));
      setTempValue(null);
    }
  };

  const handleBlur: React.FocusEventHandler<HTMLInputElement> = (e) => {
    console.log("ABC");
    handleChange(e.target.value, true);
    setTempValue(null);
    onBlur?.(e);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>): void => {
    console.log("DEF");
    if (!onlyChangeOnBlur) return;
    if (e.key === "Enter") e.currentTarget.blur();
  };

  const parsedValue = new TimeStamp(value, "UTC").fString("ISO", "local").slice(0, -1);

  return (
    <Text
      type="datetime-local"
      onBlur={handleBlur}
      required={false}
      onKeyDown={handleKeyDown}
      value={tempValue ?? parsedValue}
      onChange={handleChange}
      {...props}
    />
  );
};
