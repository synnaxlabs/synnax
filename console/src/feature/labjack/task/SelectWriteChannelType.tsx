// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/labjack/task/SelectWriteChannelType.css";

import { Select } from "@synnaxlabs/lyra/select";

import { type WriteChannelType } from "@/feature/labjack/task/types";
import { CSS } from "@/platform/css";

export interface SelectWriteChannelTypeProps extends Select.ButtonsProps<WriteChannelType> {}

export const SelectWriteChannelType = (props: SelectWriteChannelTypeProps) => (
  <Select.Buttons {...props}>
    <Select.Item itemKey="analog" className={CSS.BE("labjack-write-type", "ao")}>
      Analog
    </Select.Item>
    <Select.Item itemKey="digital">Digital</Select.Item>
  </Select.Buttons>
);
