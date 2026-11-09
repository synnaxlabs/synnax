// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";

import { type WriteChannelType } from "@/feature/labjack/task/types";

const TRIGGER_PROPS: Select.SingleTriggerProps = { variant: "shadow" };

export interface SelectWriteChannelTypeProps extends Omit<
  Select.SingleSimpleProps<WriteChannelType>,
  "resourceName" | "children"
> {}

export const SelectWriteChannelType = (props: SelectWriteChannelTypeProps) => (
  <Select.Simple<WriteChannelType>
    resourceName="channel type"
    allowNone={false}
    triggerProps={TRIGGER_PROPS}
    {...props}
  >
    <Select.Item itemKey="analog">Analog</Select.Item>
    <Select.Item itemKey="digital">Digital</Select.Item>
  </Select.Simple>
);
