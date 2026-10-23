// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { simulator } from "@/fixtures/simulator";
import { type capture } from "@/index";

import { OPCUA, openTaskForm } from "./common/server";

export const driver = true;

const NODES = ["command_0", "command_1"];

/**
 * Docs `reference/hardware/opc-ua/write-task` (Create an OPC UA write task): create a
 * write task from a simulated OPC UA server, drag two writable variables from the
 * browser into the channels, and start the task.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const sim = await simulator(OPCUA.server);
  try {
    const form = await openTaskForm(session, OPCUA, "Create write task");

    await session.zoom(form);
    const browser = form.locator(".console-opc-browser").first();
    const node = (name: string) =>
      browser.getByRole("treeitem").filter({ hasText: new RegExp(`^${name}$`) });
    await session.waitFor(node("MyObject"));
    await session.click(node("MyObject"), { text: true });
    const list = form.locator(".console-channel-list").first();
    for (const name of NODES) {
      await session.waitFor(node(name));
      await session.hold(400);
      await session.drag(node(name), list);
      await session.waitFor(list.getByRole("option").filter({ hasText: name }));
      await session.hold(500);
    }

    await session.click(form.locator("button:has(svg.pluto-icon--play)").first());
    session.endZoom();
    await session.waitFor(form.getByText("Task started successfully").first());
    await session.moveTo({ x: 1100, y: 120 });
    await session.hold(3000);
  } finally {
    await sim.stop();
  }
};
