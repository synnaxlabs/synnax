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

import { type ReadChannelType } from "@/feature/modbus/task/types";

export type SelectReadChannelTypeFieldProps = Form.SelectFieldProps<ReadChannelType>;

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
    children: (
      <>
        <Select.Item itemKey="coil">Coil</Select.Item>
        <Select.Item itemKey="discrete_input">Discrete</Select.Item>
        <Select.Item itemKey="holding_register">Holding register</Select.Item>
        <Select.Item itemKey="input_register">Register</Select.Item>
      </>
    ),
  },
});
