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

import { type WriteChannelType } from "@/feature/modbus/task/types";

export interface WriteChannelTypeEntry extends record.KeyedNamed<WriteChannelType> {}

const NAMES: Record<WriteChannelType, string> = {
  coil: "Coil",
  holding_register: "Holding register",
};

const DATA: WriteChannelTypeEntry[] = modbus.WRITE_CHANNEL_TYPES.map((key) => ({
  key,
  name: NAMES[key],
}));

export interface SelectWriteChannelTypeFieldProps extends Omit<
  Form.SelectFieldProps<WriteChannelType, WriteChannelTypeEntry>,
  "data" | "entryRenderKey" | "columns"
> {}

export const SelectWriteChannelTypeField = Form.buildSelectField<
  WriteChannelType,
  WriteChannelTypeEntry
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
