// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { OPCUA, serverTaskScript } from "./common/server";

export const driver = true;

const NODES = ["my_float_0", "my_float_1"];

/**
 * Docs `reference/hardware/opc-ua/read-task` (Create an OPC UA read task): create a
 * read task from a simulated OPC UA server, set the rates, drag two variables from the
 * browser into the channels, and start the task.
 */
export default serverTaskScript(OPCUA, {
  item: "Create read task",
  rates: { "Sample rate": "100", "Stream rate": "20" },
  channels: async (session, form) => {
    const browser = form.locator(".console-opc-browser").first();
    const node = (name: string): Locator =>
      browser.getByRole("treeitem").filter({ hasText: new RegExp(`^${name}$`) });
    await session.waitFor(node("MyObject"));
    await session.click(node("MyObject"), { text: true });
    const list = form.locator(".console-channel-list").first();
    for (const name of NODES) {
      await session.waitFor(node(name));
      await node(name).scrollIntoViewIfNeeded();
      await session.hold(400);
      await session.drag(node(name), list);
      await session.waitFor(list.getByRole("option").filter({ hasText: name }));
      await session.hold(500);
    }
  },
});
