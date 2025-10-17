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
  CI_CHANNEL_SCHEMAS,
  CI_CHANNEL_TYPE_ICONS,
  CI_CHANNEL_TYPE_NAMES,
  type CIChannel,
  type CIChannelType,
  ZERO_CI_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends record.KeyedNamed<CIChannelType> {}

export type SelectCIChannelTypeFieldProps = Form.SelectFieldProps<CIChannelType, Entry>;

export const SelectCIChannelTypeField = Form.buildSelectField<CIChannelType, Entry>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel Type",
    onChange: (value, { get, set, path }) => {
      const prevType = get<CIChannelType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_CI_CHANNELS[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<CIChannel>(parentPath).value;
      const schema = CI_CHANNEL_SCHEMAS[value];
      const nextValue = {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
        ...("minVal" in next && { minVal: next.minVal }),
        ...("maxVal" in next && { maxVal: next.maxVal }),
      };
      set(parentPath, nextValue);
    },
  },
  inputProps: {
    resourceName: "Channel Type",
    data: Object.keys(CI_CHANNEL_TYPE_NAMES).map((key) => {
      const type = key as CIChannelType;
      const Icon = CI_CHANNEL_TYPE_ICONS[type];
      return {
        key: type,
        name: CI_CHANNEL_TYPE_NAMES[type],
        icon: <Icon color={8} />,
      };
    }),
  },
});
