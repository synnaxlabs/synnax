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
} from "@/feature/ni/task/types";

export const SelectAOChannelTypeField = Form.buildSelectField<AOChannelType>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel type",
    onChange: (value, { get, set, path }) => {
      const prevType = get<AOChannelType>(path).value;
      if (prevType === value) return;
      const next = createAOChannel(value);
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
    children: AO_CHANNEL_TYPES.map((key) => {
      const Icon = AO_CHANNEL_TYPE_ICONS[key];
      return (
        <Select.Item key={key} itemKey={key}>
          <Icon color={9} />
          {AO_CHANNEL_TYPE_NAMES[key]}
        </Select.Item>
      );
    }),
  },
});
