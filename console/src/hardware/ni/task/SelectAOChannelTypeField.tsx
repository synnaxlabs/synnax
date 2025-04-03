// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Text } from "@synnaxlabs/pluto";
import { deep, type Keyed } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import {
  AO_CHANNEL_SCHEMAS,
  AO_CHANNEL_TYPE_ICONS,
  AO_CHANNEL_TYPE_NAMES,
  type AOChannel,
  type AOChannelType,
  ZERO_AO_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends Keyed<AOChannelType> {
  name: ReactElement;
}

interface ChannelTypeProps {
  type: AOChannelType;
}

const ChannelType = ({ type }: ChannelTypeProps) => (
  <Text.WithIcon startIcon={AO_CHANNEL_TYPE_ICONS[type]} level="p">
    {AO_CHANNEL_TYPE_NAMES[type]}
  </Text.WithIcon>
);

export type SelectAOChannelTypeFieldProps = Form.DropdownButtonFieldProps<
  AOChannelType,
  Entry
>;

export const SelectAOChannelTypeField = Form.buildDropdownButtonSelectField<
  AOChannelType,
  Entry
>({
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
    entryRenderKey: "name",
    columns: [
      {
        key: "name",
        name: "Name",
        render: ({ entry: { key } }) => <ChannelType type={key} />,
      },
    ],
    data: Object.keys(AO_CHANNEL_TYPE_NAMES).map((key) => ({
      key: key as AOChannelType,
      name: <ChannelType type={key as AOChannelType} />,
    })),
  },
});
