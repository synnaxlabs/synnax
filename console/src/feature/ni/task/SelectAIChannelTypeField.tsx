// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { deep, record } from "@synnaxlabs/x";

import {
  AI_CHANNEL_SCHEMAS,
  AI_CHANNEL_TYPE_ICONS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  createAIChannel,
} from "@/feature/ni/task/types";

/** The items of the channel type select, one for each of the types. */
export const channelTypeItems = (types: readonly AIChannelType[]) =>
  types.map((type) => {
    const Icon = AI_CHANNEL_TYPE_ICONS[type];
    return (
      <Select.Item key={type} itemKey={type}>
        <Icon color={9} />
        {AI_CHANNEL_TYPE_NAMES[type]}
      </Select.Item>
    );
  });

const SelectType = Form.buildSelectField<AIChannelType>({
  fieldProps: { label: "Channel type" },
  inputProps: {
    resourceName: "channel type",
    children: channelTypeItems(record.keys(AI_CHANNEL_TYPE_NAMES)),
  },
});

interface SelectAIChannelTypeFieldProps extends Pick<
  Form.SelectFieldProps<AIChannelType>,
  "inputProps"
> {
  path: deep.Path<AIChannel>;
}

export const SelectAIChannelTypeField = ({
  path,
  inputProps,
}: SelectAIChannelTypeFieldProps) => (
  <SelectType
    path={path.type}
    inputProps={inputProps}
    onChange={(value, { get, set }) => {
      const prevType = get(path.type).value;
      if (prevType === value) return;
      const next = createAIChannel(value);
      const prevParent = get(path).value;
      const schema = AI_CHANNEL_SCHEMAS[value];
      set(path, {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
      } as AIChannel);
    }}
  />
);
