// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input } from "@synnaxlabs/lyra/input";
import { type telem, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

// SY-4816: Static range entry, when added, should stop at 1s, which also simplifies
// its selector and UX.
export interface CustomInputProps extends Omit<
  Input.ItemProps,
  "label" | "onChange" | "children"
> {
  value: telem.Window | undefined;
  onChange: (value: telem.Window) => void;
  autoFocus?: boolean;
  disabled?: boolean;
}

/** Edits the rolling window the "custom" range key resolves to. */
export const CustomInput = ({
  value,
  onChange,
  autoFocus,
  disabled,
  ...rest
}: CustomInputProps): ReactElement => {
  const handleChange = useCallback(
    (raw: string) => {
      const span = TimeSpan.parse(raw);
      if (span == null || span.isZero) return;
      onChange({ variant: "dynamic", timeSpan: Number(span) });
    },
    [onChange],
  );
  const text =
    value?.variant === "dynamic" ? new TimeSpan(value.timeSpan).toString() : "";
  return (
    <Input.Item x label="Custom" {...rest}>
      <Input.Text
        value={text}
        onChange={handleChange}
        onlyChangeOnBlur
        resetOnBlurIfEmpty
        placeholder="1h 30min"
        autoFocus={autoFocus}
        disabled={disabled}
        grow
      />
    </Input.Item>
  );
};
