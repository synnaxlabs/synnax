// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import {
  AO_CHANNEL_TYPE,
  DO_CHANNEL_TYPE,
  type OutputChannelType,
} from "@/hardware/labjack/task/types";

export interface OutputChannelTypeEntry extends record.KeyedNamed<OutputChannelType> {}

const DATA: OutputChannelType[] = [AO_CHANNEL_TYPE, DO_CHANNEL_TYPE];

export interface SelectOutputChannelTypeProps
  extends Select.SingleProps<OutputChannelType> {}

export const SelectOutputChannelType = ({
  value,
  onChange,
}: SelectOutputChannelTypeProps) => {
  const selectProps = Select.useSingle({ data: DATA, value, onChange });
  return (
    <Select.Buttons value={value} {...selectProps}>
      <Select.Button itemKey={AO_CHANNEL_TYPE}>Analog</Select.Button>
      <Select.Button itemKey={DO_CHANNEL_TYPE}>Digital</Select.Button>
    </Select.Buttons>
  );
};
