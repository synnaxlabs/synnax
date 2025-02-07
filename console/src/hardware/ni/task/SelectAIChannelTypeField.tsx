// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, type List } from "@synnaxlabs/pluto";
import { deep, type KeyedNamed } from "@synnaxlabs/x";

import {
  ANALOG_INPUT_CHANNEL_SCHEMAS,
  ANALOG_INPUT_CHANNEL_TYPE_NAMES,
  type AnalogInputChannel,
  type AnalogInputChannelType,
  ZERO_ANALOG_INPUT_CHANNELS,
} from "@/hardware/ni/task/types";

const NAMED_KEY_COLS: List.ColumnSpec<string, KeyedNamed>[] = [
  { key: "name", name: "Name" },
];

export type SelectAIChannelTypeFieldProps = Form.SelectSingleFieldProps<
  AnalogInputChannelType,
  KeyedNamed<AnalogInputChannelType>
>;

export const SelectAIChannelTypeField = Form.buildSelectSingleField<
  AnalogInputChannelType,
  KeyedNamed<AnalogInputChannelType>
>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel Type",
    onChange: (value, { get, set, path }) => {
      const prevType = get<AnalogInputChannelType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_ANALOG_INPUT_CHANNELS[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<AnalogInputChannel>(parentPath).value;
      const schema = ANALOG_INPUT_CHANNEL_SCHEMAS[value];
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
      });
    },
  },
  inputProps: {
    hideColumnHeader: true,
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: (
      Object.entries(ANALOG_INPUT_CHANNEL_TYPE_NAMES) as [
        AnalogInputChannelType,
        string,
      ][]
    ).map(([key, name]) => ({ key, name })),
  },
});
