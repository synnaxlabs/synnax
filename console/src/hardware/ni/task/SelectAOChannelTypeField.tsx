// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, type List } from "@synnaxlabs/pluto";
import { deep, type KeyedNamed } from "@synnaxlabs/x";
import { type z } from "zod";

import {
  AO_CHANNEL_SCHEMAS,
  AO_CHANNEL_TYPE_NAMES,
  type AOChannel,
  type AOChannelType,
  ZERO_AO_CHANNELS,
} from "@/hardware/ni/task/types";

const NAMED_KEY_COLS: List.ColumnSpec<string, KeyedNamed>[] = [
  { key: "name", name: "Name" },
];

export const SelectAOChannelTypeField = Form.buildSelectSingleField<
  AOChannelType,
  KeyedNamed<AOChannelType>
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
      let schema = AO_CHANNEL_SCHEMAS[value];
      if ("sourceType" in schema)
        // @ts-expect-error - schema source type checking
        schema = schema.sourceType() as z.ZodObject<AOChannel>;
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
      });
    },
  },
  inputProps: {
    hideColumnHeader: true,
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: (Object.entries(AO_CHANNEL_TYPE_NAMES) as [AOChannelType, string][]).map(
      ([key, name]) => ({ key, name }),
    ),
  },
});
