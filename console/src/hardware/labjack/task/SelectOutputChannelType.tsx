// Copyright 2026 Synnax Labs, Inc.
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

export interface SelectOutputChannelTypeProps extends Omit<
  Select.ButtonsProps<OutputChannelType>,
  "keys"
> {}

export const SelectOutputChannelType = (props: SelectOutputChannelTypeProps) => (
  <Select.Buttons {...props} keys={DATA}>
    <Select.Button itemKey={AO_CHANNEL_TYPE} style={{ borderRadius: 0 }}>
      Analog
    </Select.Button>
    <Select.Button itemKey={DO_CHANNEL_TYPE}>Digital</Select.Button>
  </Select.Buttons>
);
