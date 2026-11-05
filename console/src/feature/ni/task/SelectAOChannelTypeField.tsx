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
import { deep } from "@synnaxlabs/x";

import {
  AO_CHANNEL_SCHEMAS,
  AO_CHANNEL_TYPE_ICONS,
  AO_CHANNEL_TYPE_NAMES,
  AO_CHANNEL_TYPES,
  type AOChannel,
  type AOChannelType,
  createAOChannel,
  isGenerated,
} from "@/feature/ni/task/types";
import { Task } from "@/platform/task";

/** The items of the channel type select, one for each of the types. */
export const channelTypeItems = (types: readonly AOChannelType[]) =>
  types.map((type) => {
    const Icon = AO_CHANNEL_TYPE_ICONS[type];
    return (
      <Select.Item key={type} itemKey={type}>
        <Icon color={9} />
        {AO_CHANNEL_TYPE_NAMES[type]}
      </Select.Item>
    );
  });

const SelectType = Form.buildSelectField<AOChannelType>({
  fieldProps: { label: "Channel type" },
  inputProps: {
    allowNone: false,
    resourceName: "channel type",
    children: channelTypeItems(AO_CHANNEL_TYPES),
  },
});

interface SelectAOChannelTypeFieldProps extends Pick<
  Form.SelectFieldProps<AOChannelType>,
  "inputProps"
> {
  path: deep.Path<AOChannel>;
}

export const SelectAOChannelTypeField = ({
  path,
  inputProps,
}: SelectAOChannelTypeFieldProps) => (
  <SelectType
    path={path.type}
    inputProps={inputProps}
    onChange={(value, { get, set }) => {
      const prevType = get(path.type).value;
      if (prevType === value) return;
      const next = createAOChannel(value);
      const prevParent = get(path).value;
      const schema = AO_CHANNEL_SCHEMAS[value];
      set(path, {
        ...deep.overrideValidItems(next, prevParent, schema),
        ...(isGenerated(next) ? Task.WRITE_CHANNEL_OVERRIDE : null),
        type: next.type,
      } as AOChannel);
    }}
  />
);
