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
  AI_CHANNEL_SCHEMAS,
  AI_CHANNEL_TYPE_ICONS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  ZERO_AI_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends record.KeyedNamed<AIChannelType> {}

export type SelectAIChannelTypeFieldProps = Form.SelectFieldProps<AIChannelType, Entry>;

export const SelectAIChannelTypeField = Form.buildSelectField<AIChannelType, Entry>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel Type",
    onChange: (value, { get, set, path }) => {
      const prevType = get<AIChannelType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_AI_CHANNELS[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<AIChannel>(parentPath).value;
      const schema = AI_CHANNEL_SCHEMAS[value];
      const nextValue = {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
      };
      set(parentPath, nextValue);
    },
  },
  inputProps: {
    resourceName: "channel type",
    data: Object.keys(AI_CHANNEL_TYPE_NAMES).map((key) => {
      const type = key as AIChannelType;
      const Icon = AI_CHANNEL_TYPE_ICONS[type];
      return {
        key: type,
        name: AI_CHANNEL_TYPE_NAMES[type],
        icon: <Icon color={8} />,
      };
    }),
  },
});
