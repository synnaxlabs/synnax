// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { modbus } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { type InputChannelType } from "@/feature/modbus/task/types";

export interface InputChannelTypeEntry extends record.KeyedNamed<InputChannelType> {}

const NAMES: Record<InputChannelType, string> = {
  coil_input: "Coil",
  discrete_input: "Discrete",
  holding_register_input: "Holding Register",
  register_input: "Register",
};

const DATA: InputChannelTypeEntry[] = modbus.INPUT_CHANNEL_TYPES.map((key) => ({
  key,
  name: NAMES[key],
}));

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
