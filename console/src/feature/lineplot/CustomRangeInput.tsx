// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Input } from "@synnaxlabs/lyra/input";
import { LinePlot } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

// SY-4816: Static range entry, when added, should stop at 1s, which also simplifies
// its selector and UX.
/** Parses a positive duration like "45min" or "1h 30m" into nanoseconds. */
export const parseSpan = (input: string): number | null => {
  const span = TimeSpan.parse(input);
  return span == null || span.isZero ? null : Number(span);
};

/** Edits the rolling window the plot's "custom" range key resolves to. */
export const CustomRangeInput = (): ReactElement => {
  const custom = LinePlot.useCustomRange();
  const dispatch = LinePlot.useSingleDispatch();
  const handleChange = useCallback(
    (raw: string) => {
      const span = parseSpan(raw);
      if (span == null) return;
      dispatch(lineplot.setCustomRange({ custom: { variant: "dynamic", span } }));
    },
    [dispatch],
  );
  const value =
    custom?.variant === "dynamic" ? new TimeSpan(custom.span).toString() : "";
  return (
    <Input.Item x label="Custom">
      <Input.Text
        value={value}
        onChange={handleChange}
        onlyChangeOnBlur
        resetOnBlurIfEmpty
        placeholder="1h 30m"
        grow
      />
    </Input.Item>
  );
};
