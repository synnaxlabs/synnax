// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax, type task } from "@synnaxlabs/client";

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPC } from "@/feature/opc";
import { PagerDuty } from "@/feature/pagerduty";
import { type Layout } from "@/platform/layout";
import { type Layout as TaskLayout } from "@/platform/task/Form";
import { type Session } from "@/session";

const ZERO_LAYOUTS: Record<string, TaskLayout> = {
  ...EtherCAT.Task.ZERO_LAYOUTS,
  ...HTTP.Task.ZERO_LAYOUTS,
  ...LabJack.Task.ZERO_LAYOUTS,
  ...Modbus.Task.ZERO_LAYOUTS,
  ...NI.Task.ZERO_LAYOUTS,
  ...OPC.Task.ZERO_LAYOUTS,
  ...PagerDuty.Task.ZERO_LAYOUTS,
};

export const createLayout = ({
  key,
  name,
  type,
}: task.Task): Session.Layout.BaseState => {
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
