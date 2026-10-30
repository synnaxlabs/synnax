// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input } from "@synnaxlabs/lyra/input";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface CustomRangeInputProps {
  value: TimeSpan;
  onChange: (span: TimeSpan) => void;
}

// SY-4816: Static range entry, when added, should stop at 1s, which also simplifies
// its selector and UX.
/** Edits the span of a plot's custom rolling window. */
export const CustomRangeInput = ({
  value,
  onChange,
}: CustomRangeInputProps): ReactElement => {
  const handleChange = (raw: string) => {
    const span = TimeSpan.parse(raw);
    if (span == null || span.isZero) return;
    onChange(span);
  };
  return (
    <Input.Item x label="Custom">
      <Input.Text
        value={value.toString()}
        onChange={handleChange}
        onlyChangeOnBlur
        resetOnBlurIfEmpty
        placeholder="1h 30min"
        grow
      />
    </Input.Item>
  );
};
