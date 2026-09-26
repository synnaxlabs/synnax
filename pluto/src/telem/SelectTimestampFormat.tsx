// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/telem/SelectTimestampFormat.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { Select } from "@synnaxlabs/lyra/select";
import { type TimestampFormat } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface SelectTimestampFormatProps extends Omit<
  Select.SimpleProps<TimestampFormat>,
  "children" | "resourceName"
> {}

export const SelectTimestampFormat = ({
  dialogProps,
  ...rest
}: SelectTimestampFormatProps): ReactElement => (
  <Select.Simple<TimestampFormat>
    {...rest}
    dialogProps={{
      ...dialogProps,
      className: CSS.cls(
        CSS.BE("select-timestamp-format", "dialog"),
        dialogProps?.className,
      ),
    }}
    resourceName="timestamp format"
  >
    <Select.Item itemKey="ISO">ISO 8601</Select.Item>
    <Select.Item itemKey="ISODate">ISO date</Select.Item>
    <Select.Item itemKey="time">Time</Select.Item>
    <Select.Item itemKey="preciseTime">Precise time</Select.Item>
    <Select.Item itemKey="date">Date</Select.Item>
    <Select.Item itemKey="dateTime">Date + Time</Select.Item>
    <Select.Item itemKey="preciseDate">Precise date</Select.Item>
  </Select.Simple>
);
