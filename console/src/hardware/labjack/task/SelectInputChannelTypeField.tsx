// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";

import { type InputChannelType } from "@/hardware/labjack/task/types";

interface InputChannelTypeEntry extends KeyedNamed<InputChannelType> {}

const INPUT_CHANNEL_TYPES: InputChannelTypeEntry[] = [
  { key: "AI", name: "Analog In" },
  { key: "DI", name: "Digital In" },
  { key: "TC", name: "Thermocouple" },
];

export const SelectInputChannelTypeField = Form.buildDropdownButtonSelectField<
  InputChannelType,
  InputChannelTypeEntry
>({
  fieldKey: "type",
  fieldProps: { label: "Channel Type" },
  inputProps: {
    allowNone: false,
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: INPUT_CHANNEL_TYPES,
  },
});
