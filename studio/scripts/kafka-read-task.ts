// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { pickValues, submitPicks } from "./common/picker";
import { KAFKA, serverTaskScript } from "./common/server";
import { setText } from "./common/task";

export const driver = true;
export const integrations = ["modbus"];

/**
 * Docs `reference/hardware/kafka/read-task` (Create a Kafka read task): enter the
 * topic, open Add data, select the oven records and two of their fields, add them, and
 * start the task. The fake broker publishes these records, so the task reads real
 * values.
 */
export default serverTaskScript(KAFKA, {
  item: "Create read task",
  properties: async (session, form) => {
    await setText(session, form, "Topic", "sensor-readings");
  },
  channels: async (session, form) => {
    const { page } = session;
    await session.click(form.getByRole("button", { name: "Add data", exact: true }));
    const dialog = page
      .locator(capture.MODAL)
      .filter({
        has: page.getByRole("button", { name: "Add an empty record", exact: true }),
      })
      .first();
    await session.waitFor(dialog);
    await session.zoom(dialog);
    const oven = dialog.getByRole("treeitem").getByText("oven", { exact: true });
    await session.waitFor(oven);
    await session.hold(1200);
    await session.click(oven, { text: true });
    await pickValues(session, dialog, ["/temperature", "/pressure"]);
    await submitPicks(session, dialog, form, "Add 2 fields");
  },
});
