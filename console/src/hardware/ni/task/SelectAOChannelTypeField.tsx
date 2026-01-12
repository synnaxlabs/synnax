// Copyright 2026 Synnax Labs, Inc.
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
  AO_CHANNEL_SCHEMAS,
  AO_CHANNEL_TYPE_ICONS,
  AO_CHANNEL_TYPE_NAMES,
  AO_CHANNEL_TYPES,
  type AOChannel,
  type AOChannelType,
  ZERO_AO_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends record.KeyedNamed<AOChannelType> {}

export type SelectAOChannelTypeFieldProps = Form.SelectFieldProps<AOChannelType, Entry>;

export const SelectAOChannelTypeField = Form.buildSelectField<AOChannelType, Entry>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel Type",
    onChange: (value, { get, set, path }) => {
      const prevType = get<AOChannelType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_AO_CHANNELS[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<AOChannel>(parentPath).value;
      const schema = AO_CHANNEL_SCHEMAS[value];
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
      });
    },
  },
  inputProps: {
    allowNone: false,
    resourceName: "channel type",
    data: AO_CHANNEL_TYPES.map((key) => {
      const Icon = AO_CHANNEL_TYPE_ICONS[key];
      return {
        key,
        name: AO_CHANNEL_TYPE_NAMES[key],
        icon: <Icon color={8} />,
      };
    }) as Entry[],
  },
});
