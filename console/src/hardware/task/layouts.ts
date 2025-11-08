// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax, type task } from "@synnaxlabs/client";

import { type Common } from "@/hardware/common";
import { LabJack } from "@/hardware/labjack";
import { Modbus } from "@/hardware/modbus";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Sequence } from "@/hardware/task/sequence";
import { type Layout } from "@/layout";

const ZERO_LAYOUTS: Record<string, Common.Task.Layout> = {
  [LabJack.Task.READ_TYPE]: LabJack.Task.READ_LAYOUT,
  [LabJack.Task.WRITE_TYPE]: LabJack.Task.WRITE_LAYOUT,
  [Modbus.Task.READ_TYPE]: Modbus.Task.READ_LAYOUT,
  [Modbus.Task.WRITE_TYPE]: Modbus.Task.WRITE_LAYOUT,
  [NI.Task.ANALOG_READ_TYPE]: NI.Task.ANALOG_READ_LAYOUT,
  [NI.Task.ANALOG_WRITE_TYPE]: NI.Task.ANALOG_WRITE_LAYOUT,
  [NI.Task.COUNTER_READ_TYPE]: NI.Task.COUNTER_READ_LAYOUT,
  [NI.Task.COUNTER_WRITE_TYPE]: NI.Task.COUNTER_WRITE_LAYOUT,
  [NI.Task.DIGITAL_READ_TYPE]: NI.Task.DIGITAL_READ_LAYOUT,
  [NI.Task.DIGITAL_WRITE_TYPE]: NI.Task.DIGITAL_WRITE_LAYOUT,
  [OPC.Task.READ_TYPE]: OPC.Task.READ_LAYOUT,
  [OPC.Task.WRITE_TYPE]: OPC.Task.WRITE_LAYOUT,
  [Sequence.TYPE]: Sequence.LAYOUT,
};

export const createLayout = ({ key, name, type }: task.Task): Layout.BaseState => {
  const baseLayout = ZERO_LAYOUTS[type];
  if (baseLayout == null) throw new Error(`No layout configured for ${type}`);
  return { ...baseLayout, key, name, args: { taskKey: key } };
};

export const retrieveAndPlaceLayout = async (
  client: Synnax | null,
  key: task.Key,
  placeLayout: Layout.Placer,
) => {
  if (client == null) throw new DisconnectedError();
  const t = await client.hardware.tasks.retrieve({ key });
  const layout = createLayout(t);
  if (t.snapshot)
    layout.tab = {
      ...layout.tab,
      editable: false,
    };
  placeLayout(layout);
};
