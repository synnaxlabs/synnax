// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { taskScript, VALVE_OUTPUTS } from "./common/task";

/**
 * Docs `reference/hardware/ethercat/write-task` (Create a write task): the task starts
 * with a channel for each output of the module. Set the rates, select a channel to see
 * its entry, remove a channel, and start the task.
 */
export default taskScript({
  device: VALVE_OUTPUTS,
  item: "Create write task",
  rates: { "Execution rate": "1000", "State update rate": "50" },
  channels: async (session, form) => {
    const channel = (name: string) => form.getByText(name, { exact: true }).first();
    await session.waitFor(channel("do1_Output_7030_01_cmd"));
    await session.hold(600);
    await session.click(channel("do1_Output_7000_01_cmd"), { text: true });
    await session.hold(1500);
    await capture.contextMenu(session, channel("do1_Output_7030_01_cmd"), "Remove");
    await session.waitForHidden(channel("do1_Output_7030_01_cmd"));
    await session.hold(800);
  },
});
