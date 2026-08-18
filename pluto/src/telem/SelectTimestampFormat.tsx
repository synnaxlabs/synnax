// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/telem/SelectTimestampFormat.css";

import { type TimestampFormat } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Select } from "@/select";

const DATA: Select.StaticEntry<TimestampFormat>[] = [
  { key: "ISO", name: "ISO 8601" },
  { key: "ISODate", name: "ISO date" },
  { key: "time", name: "Time" },
  { key: "preciseTime", name: "Precise time" },
  { key: "date", name: "Date" },
  { key: "dateTime", name: "Date + Time" },
  { key: "preciseDate", name: "Precise date" },
];

export interface SelectTimestampFormatProps extends Omit<
  Select.StaticProps<TimestampFormat>,
  "data" | "resourceName"
> {}

export const SelectTimestampFormat = ({
  dialogProps,
  ...rest
}: SelectTimestampFormatProps): ReactElement => (
  <Select.Static
    {...rest}
    dialogProps={{
      ...dialogProps,
      className: CSS.cx(
        CSS.BE("select-timestamp-format", "dialog"),
        dialogProps?.className,
      ),
    }}
    data={DATA}
    resourceName="timestamp format"
  />
);
