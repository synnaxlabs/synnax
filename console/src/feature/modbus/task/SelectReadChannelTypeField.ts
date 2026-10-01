// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";

import { type ReadChannelType } from "@/feature/modbus/task/types";
import { Task } from "@/platform/task";

const NAMES = {
  coil: "Coil",
  discrete_input: "Discrete",
  holding_register: "Holding register",
  input_register: "Register",
} as const satisfies Record<ReadChannelType, string>;

export const SelectReadChannelTypeField = Form.buildSelectField<ReadChannelType>({
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
