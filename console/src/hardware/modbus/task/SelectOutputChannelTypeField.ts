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
  COIL_OUTPUT_TYPE,
  HOLDING_REGISTER_OUTPUT_TYPE,
  type OutputChannelType,
} from "@/hardware/modbus/task/types";

export interface OutputChannelTypeEntry extends record.KeyedNamed<OutputChannelType> {}

const OUTPUT_CHANNEL_TYPES: OutputChannelTypeEntry[] = [
  { key: COIL_OUTPUT_TYPE, name: "Coil" },
  { key: HOLDING_REGISTER_OUTPUT_TYPE, name: "Holding Register" },
];

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
    data: OUTPUT_CHANNEL_TYPES,
    style: { width: "25rem" },
  },
});
