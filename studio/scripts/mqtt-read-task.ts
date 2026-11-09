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
import { choose } from "./common/task";

export const driver = true;
export const integrations = ["modbus"];

/**
 * Docs `reference/hardware/mqtt/read-task` (Create an MQTT read task): open Add data,
 * select a topic of the simulated broker and two of its fields, add them as channels,
 * change the data type of one field, and start the task.
 */
export default serverTaskScript(MQTT, {
  item: "Create read task",
  channels: async (session, form) => {
    const dialog = await openDiscover(session, form, "Add data");
    const topic = dialog.getByRole("treeitem").getByText("sensors", { exact: true });
    await session.waitFor(topic);
    await session.hold(1200);
    await session.click(topic, { text: true });
    await pickValues(session, dialog, ["temperature", "pressure"]);
    await submitDiscover(session, dialog, form, "Add 2 channels");

    await session.click(
      form.getByRole("treeitem").getByText("/temperature", { exact: true }),
      { text: true },
    );
    await session.hold(800);
    await choose(session, form, "Data type", "float32", { search: true });
  },
});
