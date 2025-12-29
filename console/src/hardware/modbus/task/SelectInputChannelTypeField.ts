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

export interface InputChannelTypeEntry extends record.KeyedNamed<InputChannelType> {}

const DATA: InputChannelTypeEntry[] = [
  { key: COIL_INPUT_TYPE, name: "Coil" },
  { key: DISCRETE_INPUT_TYPE, name: "Discrete" },
  { key: HOLDING_REGISTER_INPUT_TYPE, name: "Holding Register" },
  { key: REGISTER_INPUT_TYPE, name: "Register" },
];

export interface SelectInputChannelTypeFieldProps extends Omit<
  Form.SelectFieldProps<InputChannelType, InputChannelTypeEntry>,
  "data" | "entryRenderKey" | "columns"
> {}

export const SelectInputChannelTypeField = Form.buildSelectField<
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
    resourceName: "channel type",
    data: DATA,
    style: { width: "25rem" },
  },
});
