// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type TimestampFormat } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Select } from "@/select";

const DATA: Select.StaticEntry<TimestampFormat>[] = [
  { key: "ISO", name: "ISO 8601" },
  { key: "ISODate", name: "ISO Date" },
  { key: "time", name: "Time" },
  { key: "preciseTime", name: "Precise Time" },
  { key: "date", name: "Date" },
  { key: "dateTime", name: "Date + Time" },
  { key: "preciseDate", name: "Precise Date" },
];

export interface SelectTimestampFormatProps extends Omit<
  Select.StaticProps<TimestampFormat>,
  "data" | "resourceName"
> {}

export const SelectTimestampFormat = (
  props: SelectTimestampFormatProps,
): ReactElement => (
  <Select.Static {...props} data={DATA} resourceName="timestamp format" />
);
