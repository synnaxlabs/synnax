// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { KAFKA, serverTaskScript } from "./common/server";
import { choose, setText } from "./common/task";

export const driver = true;
export const integrations = ["modbus"];

/**
 * Docs `reference/hardware/kafka/read-task` (Create a Kafka read task): enter the topic,
 * add a record with a key, add a field with a pointer and a data type, and start the
 * task. The fake broker publishes these records, so the task reads real values.
 */
export default serverTaskScript(KAFKA, {
  item: "Create read task",
  properties: async (session, form) => {
    await setText(session, form, "Topic", "sensor-readings");
  },
  channels: async (session, form) => {
    await session.click(form.getByRole("button", { name: "Add record", exact: true }));
    await session.hold(800);
    await setText(session, form, "Record key", "oven");
    const adder = form.getByRole("treeitem").getByText("Add field", { exact: true });
    await session.click(adder, { text: true });
    await session.hold(800);
    await setText(session, form, "Pointer", "/temperature");
    await choose(session, form, "Data type", "float64", { search: true });
  },
});
