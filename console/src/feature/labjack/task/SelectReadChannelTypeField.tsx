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

import { type ReadChannelType } from "@/feature/labjack/task/types";

export type SelectReadChannelTypeFieldProps = Form.SelectFieldProps<ReadChannelType>;

export const SelectReadChannelTypeField = Form.buildSelectField<ReadChannelType>({
  fieldKey: "type",
  fieldProps: { label: "Channel type" },
  inputProps: {
    allowNone: false,
    resourceName: "channel type",
    children: (
      <>
        <Select.Item itemKey="analog">Analog input</Select.Item>
        <Select.Item itemKey="digital">Digital input</Select.Item>
        <Select.Item itemKey="thermocouple">Thermocouple</Select.Item>
      </>
    ),
  },
});
