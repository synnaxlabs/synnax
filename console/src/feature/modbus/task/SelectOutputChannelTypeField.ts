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

import { type OutputChannelType } from "@/feature/modbus/task/types";

export interface OutputChannelTypeEntry extends record.KeyedNamed<OutputChannelType> {}

const NAMES: Record<OutputChannelType, string> = {
  coil_output: "Coil",
  holding_register_output: "Holding Register",
};

const DATA: OutputChannelTypeEntry[] = modbus.OUTPUT_CHANNEL_TYPES.map((key) => ({
  key,
  name: NAMES[key],
}));

export interface SelectOutputChannelTypeFieldProps extends Omit<
  Form.SelectFieldProps<OutputChannelType, OutputChannelTypeEntry>,
  "data" | "entryRenderKey" | "columns"
> {}

export const SelectOutputChannelTypeField = Form.buildSelectField<
  OutputChannelType,
  OutputChannelTypeEntry
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
