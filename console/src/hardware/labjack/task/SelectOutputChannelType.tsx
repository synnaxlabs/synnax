// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";

import {
  AO_CHANNEL_TYPE,
  DO_CHANNEL_TYPE,
  type OutputChannelType,
} from "@/hardware/labjack/task/types";

export interface OutputChannelTypeEntry extends KeyedNamed<OutputChannelType> {}

const OUTPUT_CHANNEL_TYPES: OutputChannelTypeEntry[] = [
  { key: AO_CHANNEL_TYPE, name: "Analog" },
  { key: DO_CHANNEL_TYPE, name: "Digital" },
];

export interface SelectOutputChannelTypeProps
  extends Omit<
    Select.ButtonProps<OutputChannelType, OutputChannelTypeEntry>,
    "data" | "entryRenderKey"
  > {}

export const SelectOutputChannelType = (props: SelectOutputChannelTypeProps) => (
  <Select.Button<OutputChannelType, OutputChannelTypeEntry>
    onClick={(e) => e.stopPropagation()}
    pack={false}
    size="medium"
    {...props}
    data={OUTPUT_CHANNEL_TYPES}
    entryRenderKey="name"
  />
);
