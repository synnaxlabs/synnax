// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { TANK_PRESSURES, taskScript } from "./common/task";

/**
 * Docs `reference/hardware/ethercat/read-task` (Create a read task): the task starts with
 * a channel for each input of the module. Set the rates, select a channel to read its
 * entry, remove a channel, and start the task.
 */
export default taskScript({
  device: TANK_PRESSURES,
  item: "Create read task",
  rates: { "Sample rate": "1000", "Stream rate": "25" },
  channels: async (session, form) => {
    const channel = (name: string) => form.getByText(name, { exact: true }).first();
    await session.waitFor(channel("ai1_Value_3101_02"));
    await session.hold(600);
    await session.click(channel("ai1_Value_3101_02"), { text: true });
    await session.hold(1500);
    await capture.contextMenu(session, channel("ai1_Status_3101_01"), "Remove");
    await session.waitForHidden(channel("ai1_Status_3101_01"));
    await session.hold(800);
  },
});
