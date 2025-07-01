// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import {
  COIL_INPUT_TYPE,
  DISCRETE_INPUT_TYPE,
  HOLDING_REGISTER_INPUT_TYPE,
  type InputChannelType,
  REGISTER_INPUT_TYPE,
} from "@/hardware/modbus/task/types";

const COLUMNS = [{ key: "name", name: "Name" }];

export interface InputChannelTypeEntry extends record.KeyedNamed<InputChannelType> {}

const INPUT_CHANNEL_TYPES: InputChannelTypeEntry[] = [
  { key: COIL_INPUT_TYPE, name: "Coil" },
  { key: DISCRETE_INPUT_TYPE, name: "Discrete" },
  { key: HOLDING_REGISTER_INPUT_TYPE, name: "Holding Register" },
  { key: REGISTER_INPUT_TYPE, name: "Register" },
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
  fieldProps: {
    label: "Channel Type",
    showLabel: false,
    showHelpText: false,
    hideIfNull: true,
  },
  inputProps: {
    allowNone: false,
    entryRenderKey: "name",
    columns: COLUMNS,
    data: INPUT_CHANNEL_TYPES,
    style: { width: "25rem" },
  },
});
