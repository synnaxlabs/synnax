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

import { type InputChannelType } from "@/hardware/labjack/task/types";

export interface InputChannelTypeEntry extends record.KeyedNamed<InputChannelType> {}

const INPUT_CHANNEL_TYPES: InputChannelTypeEntry[] = [
  { key: "AI", name: "Analog Input" },
  { key: "DI", name: "Digital Input" },
  { key: "TC", name: "Thermocouple" },
];

export type SelectInputChannelTypeFieldProps = Form.SelectFieldProps<
  InputChannelType,
  InputChannelTypeEntry
>;

export const SelectInputChannelTypeField = Form.buildSelectField<
  InputChannelType,
  InputChannelTypeEntry
>({
  fieldKey: "type",
  fieldProps: { label: "Channel Type" },
  inputProps: {
    allowNone: false,
    resourceName: "channel type",
    data: INPUT_CHANNEL_TYPES,
  },
});
