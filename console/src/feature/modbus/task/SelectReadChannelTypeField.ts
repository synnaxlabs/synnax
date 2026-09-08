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

import { type ReadChannelType } from "@/feature/modbus/task/types";

export interface ReadChannelTypeEntry extends record.KeyedNamed<ReadChannelType> {}

const NAMES: Record<ReadChannelType, string> = {
  coil: "Coil",
  discrete_input: "Discrete",
  holding_register: "Holding register",
  input_register: "Register",
};

const DATA: ReadChannelTypeEntry[] = modbus.READ_CHANNEL_TYPES.map((key) => ({
  key,
  name: NAMES[key],
}));

export interface SelectReadChannelTypeFieldProps extends Omit<
  Form.SelectFieldProps<ReadChannelType, ReadChannelTypeEntry>,
  "data" | "entryRenderKey" | "columns"
> {}

export const SelectReadChannelTypeField = Form.buildSelectField<
  ReadChannelType,
  ReadChannelTypeEntry
>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel type",
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
