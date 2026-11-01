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
import { type ComponentProps } from "react";

import {
  CI_CHANNEL_SCHEMAS,
  CI_CHANNEL_TYPE_ICONS,
  CI_CHANNEL_TYPE_NAMES,
  type CIChannel,
  type CIChannelType,
  createCIChannel,
} from "@/feature/ni/task/types";

/** The items of the channel type select, one for each of the types. */
export const channelTypeItems = (types: readonly CIChannelType[]) =>
  types.map((type) => {
    const Icon = CI_CHANNEL_TYPE_ICONS[type];
    return (
      <Select.Item key={type} itemKey={type}>
        <Icon color={9} />
        {CI_CHANNEL_TYPE_NAMES[type]}
      </Select.Item>
    );
  });

const SelectType = Form.buildSelectField<CIChannelType>({
  fieldProps: { label: "Channel type" },
  inputProps: {
    resourceName: "channel type",
    children: channelTypeItems(record.keys(CI_CHANNEL_TYPE_NAMES)),
  },
});

export interface SelectCIChannelTypeFieldProps extends Pick<
  ComponentProps<typeof SelectType>,
  "inputProps"
> {
  path: deep.Path<CIChannel>;
}

export const SelectCIChannelTypeField = ({
  path,
  inputProps,
}: SelectCIChannelTypeFieldProps) => (
  <SelectType
    path={path.type}
    inputProps={inputProps}
    onChange={(value, { get, set }) => {
      const prevType = get(path.type).value;
      if (prevType === value) return;
      const next = createCIChannel(value);
      const prevParent = get(path).value;
      const schema = CI_CHANNEL_SCHEMAS[value];
      set(path, {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
        ...("minVal" in next && { minVal: next.minVal }),
        ...("maxVal" in next && { maxVal: next.maxVal }),
      } as CIChannel);
    }}
  />
);
