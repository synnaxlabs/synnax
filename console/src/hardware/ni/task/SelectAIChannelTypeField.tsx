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
  AI_CHANNEL_SCHEMAS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  ZERO_AI_CHANNELS,
} from "@/hardware/ni/task/types";

const NAMED_KEY_COLS: List.ColumnSpec<string, KeyedNamed>[] = [
  { key: "name", name: "Name" },
];

export const SelectAIChannelTypeField = Form.buildSelectSingleField<
  AIChannelType,
  KeyedNamed<AIChannelType>
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
      let schema = AI_CHANNEL_SCHEMAS[value];
      if ("sourceType" in schema)
        // @ts-expect-error - schema source type checking
        schema = schema.sourceType() as z.ZodObject<AIChannel>;
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
    data: (Object.entries(AI_CHANNEL_TYPE_NAMES) as [AIChannelType, string][]).map(
      ([key, name]) => ({ key, name }),
    ),
  },
});
