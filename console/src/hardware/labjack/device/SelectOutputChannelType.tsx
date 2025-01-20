// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";

import { type OutputChannelType } from "@/hardware/labjack/device/types";

interface OutputChannelTypeEntry extends KeyedNamed<OutputChannelType> {}

const OUTPUT_CHANNEL_TYPES: OutputChannelTypeEntry[] = [
  { key: "AO", name: "Analog" },
  { key: "DO", name: "Digital" },
];

export interface SelectOutputChannelTypeProps
  extends Omit<
    Select.ButtonProps<OutputChannelType, OutputChannelTypeEntry>,
    "data" | "entryRenderKey"
  > {}

export const SelectOutputChannelType = (props: SelectOutputChannelTypeProps) => (
  <Select.Button<OutputChannelType, OutputChannelTypeEntry>
    data={OUTPUT_CHANNEL_TYPES}
    entryRenderKey="name"
    {...props}
  />
);
