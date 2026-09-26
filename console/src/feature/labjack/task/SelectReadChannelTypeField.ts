// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { labjack } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type record } from "@synnaxlabs/x";

import { type ReadChannelType } from "@/feature/labjack/task/types";

export interface ReadChannelTypeEntry extends record.KeyedNamed<ReadChannelType> {}

export const READ_CHANNEL_TYPE_NAMES: Record<ReadChannelType, string> = {
  analog: "Analog input",
  digital: "Digital input",
  thermocouple: "Thermocouple",
};

const READ_CHANNEL_TYPES: ReadChannelTypeEntry[] = labjack.readChannelTypeZ.options.map(
  (key) => ({ key, name: READ_CHANNEL_TYPE_NAMES[key] }),
);

export type SelectReadChannelTypeFieldProps = Form.SelectFieldProps<
  ReadChannelType,
  ReadChannelTypeEntry
>;

export const SelectReadChannelTypeField = Form.buildSelectField<
  ReadChannelType,
  ReadChannelTypeEntry
>({
  fieldKey: "type",
  fieldProps: { label: "Channel type" },
  inputProps: {
    allowNone: false,
    resourceName: "channel type",
    data: READ_CHANNEL_TYPES,
  },
});
