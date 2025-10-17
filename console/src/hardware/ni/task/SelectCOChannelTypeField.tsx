// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { deep, type record } from "@synnaxlabs/x";

import {
  CO_CHANNEL_SCHEMAS,
  CO_CHANNEL_TYPE_ICONS,
  CO_CHANNEL_TYPE_NAMES,
  CO_PULSE_OUTPUT_CHAN_TYPE,
  type COChannel,
  type COChannelType,
  ZERO_CO_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends record.KeyedNamed<COChannelType> {}

export type SelectCOChannelTypeFieldProps = Form.SelectFieldProps<COChannelType, Entry>;

export const SelectCOChannelTypeField = Form.buildSelectField<COChannelType, Entry>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel Type",
    onChange: (value, { get, set, path }) => {
      const prevType = get<COChannelType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_CO_CHANNELS[value]) as COChannel;
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<COChannel>(parentPath).value;
      const schema = CO_CHANNEL_SCHEMAS[value];
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
      } as COChannel);
    },
  },
  inputProps: {
    allowNone: false,
    resourceName: "Channel Type",
    data: [CO_PULSE_OUTPUT_CHAN_TYPE].map((key) => {
      const Icon = CO_CHANNEL_TYPE_ICONS[key as COChannelType];
      return {
        key,
        name: CO_CHANNEL_TYPE_NAMES[key as COChannelType],
        icon: <Icon color={8} />,
      };
    }) as Entry[],
  },
});
