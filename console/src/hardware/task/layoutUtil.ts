// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, type task } from "@synnaxlabs/client";

import { NULL_CLIENT_ERROR } from "@/errors";
import { type Common } from "@/hardware/common";
import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Sequence } from "@/hardware/task/sequence";
import { type Layout } from "@/layout";

const ZERO_LAYOUT_STATES: Record<string, Common.Task.Layout> = {
  [LabJack.Task.READ_TYPE]: LabJack.Task.READ_LAYOUT,
  [LabJack.Task.WRITE_TYPE]: LabJack.Task.WRITE_LAYOUT,
  [OPC.Task.READ_TYPE]: OPC.Task.READ_LAYOUT,
  [OPC.Task.WRITE_TYPE]: OPC.Task.WRITE_LAYOUT,
  [NI.Task.ANALOG_READ_TYPE]: NI.Task.ANALOG_READ_LAYOUT,
  [NI.Task.DIGITAL_WRITE_TYPE]: NI.Task.DIGITAL_WRITE_LAYOUT,
  [NI.Task.DIGITAL_READ_TYPE]: NI.Task.DIGITAL_READ_LAYOUT,
  [NI.Task.ANALOG_WRITE_TYPE]: NI.Task.ANALOG_WRITE_LAYOUT,
  [Sequence.TYPE]: Sequence.LAYOUT,
};

export const createLayout = ({ key, type }: task.Task): Layout.BaseState => {
  const configureLayout = ZERO_LAYOUT_STATES[type];
  if (configureLayout == null) throw new Error(`No layout configured for ${type}`);
  return { ...configureLayout, key, args: { taskKey: key } };
};

export const retrieveAndPlaceLayout = async (
  client: Synnax | null,
  key: task.Key,
  placeLayout: Layout.Placer,
) => {
  if (client == null) throw NULL_CLIENT_ERROR;
  const t = await client.hardware.tasks.retrieve(key);
  const layout = createLayout(t);
  placeLayout(layout);
};
