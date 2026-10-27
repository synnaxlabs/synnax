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
import { type ReactElement } from "react";

// SY-4816: Static range entry, when added, should stop at 1s, which also simplifies
// its selector and UX.
/** Edits the rolling window the plot's "custom" range key resolves to. */
export const CustomRangeInput = (): ReactElement => {
  const custom = LinePlot.useCustomRange();
  const dispatch = LinePlot.useSingleDispatch();
  const handleChange = (raw: string) => {
    const span = TimeSpan.parse(raw);
    if (span == null || span.isZero) return;
    dispatch(
      lineplot.setCustomRange({ custom: { variant: "dynamic", span: Number(span) } }),
    );
  };
  const value =
    custom?.variant === "dynamic" ? new TimeSpan(custom.span).toString() : "";
  return (
    <Input.Item x label="Custom">
      <Input.Text
        value={value}
        onChange={handleChange}
        onlyChangeOnBlur
        resetOnBlurIfEmpty
        placeholder="1h 30min"
        grow
      />
    </Input.Item>
  );
};
