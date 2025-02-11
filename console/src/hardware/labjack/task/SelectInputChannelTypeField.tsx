// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";

import {
  AI_CHANNEL_TYPE,
  DI_CHANNEL_TYPE,
  type InputChannelType,
  TC_CHANNEL_TYPE,
} from "@/hardware/labjack/task/types";

const COLUMNS = [{ key: "name", name: "Name" }];

export interface InputChannelTypeEntry extends KeyedNamed<InputChannelType> {}

const INPUT_CHANNEL_TYPES: InputChannelTypeEntry[] = [
  { key: AI_CHANNEL_TYPE, name: "Analog Input" },
  { key: DI_CHANNEL_TYPE, name: "Digital Input" },
  { key: TC_CHANNEL_TYPE, name: "Thermocouple" },
];

export interface SelectInputChannelTypeFieldProps
  extends Omit<
    Form.DropdownButtonFieldProps<InputChannelType, InputChannelTypeEntry>,
    "data" | "entryRenderKey" | "columns"
  > {}

export const SelectInputChannelTypeField = Form.buildDropdownButtonSelectField<
  InputChannelType,
  InputChannelTypeEntry
>({
  fieldKey: "type",
  fieldProps: { label: "Channel Type" },
  inputProps: {
    allowNone: false,
    entryRenderKey: "name",
    columns: COLUMNS,
    data: INPUT_CHANNEL_TYPES,
  },
});
