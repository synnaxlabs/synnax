// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type ReactElement } from "react";

import {
  type Model,
  T4_MODEL,
  T7_MODEL,
  T8_MODEL,
} from "@/feature/labjack/device/types";
import { type ReadChannelType } from "@/feature/labjack/task/types";
import { Task } from "@/platform/task";

export const READ_CHANNEL_TYPE_NAMES = {
  analog: "Analog input",
  digital: "Digital input",
  thermocouple: "Thermocouple",
} as const satisfies Record<ReadChannelType, string>;

const { thermocouple: _, ...T4_READ_CHANNEL_TYPE_NAMES } = READ_CHANNEL_TYPE_NAMES;

/** The channel types each model reads. The T4 has no thermocouple inputs. */
export const READ_CHANNEL_TYPE_ITEMS: Record<Model, ReactElement[]> = {
  [T4_MODEL]: Task.selectItems(T4_READ_CHANNEL_TYPE_NAMES),
  [T7_MODEL]: Task.selectItems(READ_CHANNEL_TYPE_NAMES),
  [T8_MODEL]: Task.selectItems(READ_CHANNEL_TYPE_NAMES),
};

export const SelectReadChannelTypeField = Form.buildSelectField<ReadChannelType>({
  fieldKey: "type",
  fieldProps: { label: "Channel type" },
  inputProps: {
    allowNone: false,
    resourceName: "channel type",
    children: Task.selectItems(READ_CHANNEL_TYPE_NAMES),
  },
});
