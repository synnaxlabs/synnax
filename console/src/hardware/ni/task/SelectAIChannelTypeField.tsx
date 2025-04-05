// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Form, Text } from "@synnaxlabs/pluto";
import { deep, type Keyed } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import {
  AI_CHANNEL_SCHEMAS,
  AI_CHANNEL_TYPE_ICONS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  ZERO_AI_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends Keyed<AIChannelType> {
  name: ReactElement;
}

interface ChannelTypeProps {
  type: AIChannelType;
}

const ChannelType = ({ type }: ChannelTypeProps) => (
  <Align.Space direction="x" size="small">
    <Text.WithIcon startIcon={AI_CHANNEL_TYPE_ICONS[type]} level="p" shade={7} />
    <Text.Text level="p">{AI_CHANNEL_TYPE_NAMES[type]}</Text.Text>
  </Align.Space>
);

export type SelectAIChannelTypeFieldProps = Form.DropdownButtonFieldProps<
  AIChannelType,
  Entry
>;

export const SelectAIChannelTypeField = Form.buildDropdownButtonSelectField<
  AIChannelType,
  Entry
>({
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
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
      });
    },
  },
  inputProps: {
    hideColumnHeader: true,
    entryRenderKey: "name",
    columns: [
      {
        key: "name",
        name: "Name",
        render: ({ entry: { key } }) => <ChannelType type={key} />,
      },
    ],
    data: Object.keys(AI_CHANNEL_TYPE_NAMES).map((key) => ({
      key: key as AIChannelType,
      name: <ChannelType type={key as AIChannelType} />,
    })),
  },
});
