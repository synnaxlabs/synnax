// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";

import { type WriteChannelType } from "@/feature/modbus/task/types";
import { Task } from "@/platform/task";

const NAMES = {
  coil: "Coil",
  holding_register: "Holding register",
} as const satisfies Record<WriteChannelType, string>;

export type SelectWriteChannelTypeFieldProps = Form.SelectFieldProps<WriteChannelType>;

export const SelectWriteChannelTypeField = Form.buildSelectField<WriteChannelType>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel type",
    showLabel: false,
    showHelpText: false,
    hideIfNull: true,
  },
  inputProps: {
    allowNone: false,
    resourceName: "channel type",
    style: { width: "25rem" },
    children: Task.selectItems(NAMES),
  },
});
