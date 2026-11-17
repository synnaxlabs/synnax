// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { openDiscover, pickValues, submitDiscover } from "./common/mqtt";
import { MQTT, serverTaskScript } from "./common/server";

export const driver = true;
export const integrations = ["modbus"];

/**
 * Docs `reference/hardware/mqtt/sparkplug` (Read Sparkplug B tags): open Add data,
 * switch to Sparkplug B, select the edge node of the simulated plant, select two tags,
 * add them as channels, and start the task.
 */
export default serverTaskScript(MQTT, {
  item: "Create read task",
  channels: async (session, form) => {
    const dialog = await openDiscover(session, form, "Add data");
    await session.hold(800);
    await session.click(
      dialog.getByRole("button", { name: "Sparkplug B", exact: true }),
      { text: true },
    );
    const node = dialog.getByRole("treeitem").getByText("Line1", { exact: true });
    await session.waitFor(node);
    await session.hold(1000);
    await session.click(node, { text: true });
    await pickValues(session, dialog, ["temperature", "speed"]);
    await submitDiscover(session, dialog, form, "Add 2 channels");
  },
});
