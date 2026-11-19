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

export const driver = true;
export const integrations = ["modbus"];

/**
 * Docs `reference/hardware/kafka/read-task` (Create a Kafka read task): open Add data,
 * enter the topic, select the oven records and two of their fields, add them, and start
 * the task. The fake broker publishes these records, so the task reads real values.
 */
export default serverTaskScript(KAFKA, {
  item: "Create read task",
  channels: async (session, form) => {
    const { page } = session;
    await session.click(form.getByRole("button", { name: "Add data", exact: true }));
    const topic = page.getByRole("textbox", { name: "Topic", exact: true });
    const dialog = page.locator(capture.MODAL).filter({ has: topic }).first();
    await session.waitFor(dialog);
    await session.zoom(dialog);
    await session.click(dialog.getByRole("textbox", { name: "Topic", exact: true }));
    await session.type("sensor-readings");
    await session.press("Enter");
    const oven = dialog.getByRole("treeitem").getByText("oven", { exact: true });
    await session.waitFor(oven);
    await session.hold(1200);
    await session.click(oven, { text: true });
    await pickValues(session, dialog, ["/temperature", "/pressure"]);
    await submitPicks(session, dialog, form, "Add 2 fields");
  },
});
