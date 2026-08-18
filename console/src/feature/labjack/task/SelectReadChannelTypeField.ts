// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { type ReadChannelType } from "@/feature/labjack/task/types";

export interface ReadChannelTypeEntry extends record.KeyedNamed<ReadChannelType> {}

const READ_CHANNEL_TYPES: ReadChannelTypeEntry[] = [
  { key: "analog", name: "Analog input" },
  { key: "digital", name: "Digital input" },
  { key: "thermocouple", name: "Thermocouple" },
];

export type SelectReadChannelTypeFieldProps = Form.SelectFieldProps<
  ReadChannelType,
  ReadChannelTypeEntry
>;

export const SelectReadChannelTypeField = Form.buildSelectField<
  ReadChannelType,
  ReadChannelTypeEntry
>({
  fieldKey: "type",
  fieldProps: { label: "Channel type" },
  inputProps: {
    allowNone: false,
    resourceName: "channel type",
    data: READ_CHANNEL_TYPES,
  },
});
