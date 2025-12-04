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
  ...LabJack.Task.ZERO_LAYOUTS,
  ...Modbus.Task.ZERO_LAYOUTS,
  ...NI.Task.ZERO_LAYOUTS,
  ...OPC.Task.ZERO_LAYOUTS,
  ...Sequence.ZERO_LAYOUTS,
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
  const t = await client.tasks.retrieve({ key });
  const layout = createLayout(t);
  if (t.snapshot)
    layout.tab = {
      ...layout.tab,
      editable: false,
    };
  placeLayout(layout);
};
