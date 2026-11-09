// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { openDiscover } from "./common/mqtt";
import { connect, MQTT } from "./common/server";
import { rackItem } from "./common/task";

export const viewport = { width: 1280, height: 800 };
export const driver = true;
export const integrations = ["modbus"];

/**
 * Still for the hero of the v0.61 release notes: the Add data dialog of an MQTT read
 * task lists what a simulated broker publishes, with the fields of a topic and their
 * last values.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const sim = await fixtures.simulator(MQTT.server);
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await connect(session, MQTT);
    await capture.openToolbar(session, "device");
    await capture.resizeToolbar(session, 320);
    const rack = rackItem(page, "Node 1");
    await session.waitFor(rack);
    await session.click(rack, { text: true });
    const broker = capture.treeItem(page, "device:", MQTT.name);
    await session.waitFor(broker);
    await capture.contextMenu(session, broker, "Create read task");
    session.endZoom();
    const form = page.locator(".console-task-configure").first();
    await session.waitFor(form);
    const dialog = await openDiscover(session, form, "Add data");
    session.endZoom();
    // The Core listens on wall time. Off camera, ticks pass too fast for it to answer.
    await session.settleWall(6000);
    const topic = dialog.getByRole("treeitem").getByText("sensors", { exact: true });
    await session.waitFor(topic);
    await session.click(topic, { text: true, zoom: false });
    for (const name of ["temperature", "pressure"]) {
      const check = dialog.getByRole("checkbox", {
        name: `Record ${name}`,
        exact: true,
      });
      await session.waitFor(check);
      await session.click(check, { zoom: false });
    }
    await session.moveTo({ x: 26, y: 600 });
    await session.settleWall(1000);
    await session.settle(1000);
    await session.still("window");
  } finally {
    await sim.stop();
  }
};
