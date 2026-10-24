// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { MODBUS_PLC, simulatedDriver } from "@/fixtures/hardware";
import { capture } from "@/index";

import { DRIVER } from "./common/task";

/**
 * Docs `reference/hardware/modbus/write-task` (Create a Modbus write task): right-click
 * a Modbus server in the devices toolbar and create a write task, add coils at
 * addresses 3 and 4, then start the task. The server is simulated.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const driver = await simulatedDriver({ devices: [MODBUS_PLC] });
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1500);

    await capture.openToolbar(session, "device");
    await session.hold(600);
    const rack = capture.treeItem(page, "rack:", DRIVER);
    await session.waitFor(rack);
    await session.click(rack, { text: true });
    const plc = capture.treeItem(page, "device:", MODBUS_PLC.name);
    await session.waitFor(plc);
    await session.hold(600);
    await session.rightClick(plc, { text: true });
    const menu = page.locator(".pluto-menu-context").first();
    await session.waitFor(menu);
    await session.hold(500);
    await session.click(menu.getByText("Create write task", { exact: true }).first(), {
      text: true,
    });

    const form = page.locator(".console-task-configure").first();
    await session.waitFor(form);
    await session.hold(1000);
    // The channel list is too wide for its fit to zoom, so zoom on a point in it.
    const box = await form
      .locator(".console-task-channel-form-container")
      .boundingBox();
    if (box == null) throw new Error("channel list is not on screen");
    await session.zoom({ x: box.x + box.width / 2, y: box.y + 250 }, 1.5);
    const rows = form.locator(".console-channel-item");
    const add = form.getByText("Add channel", { exact: true }).first();

    await session.click(add, { text: true });
    await session.waitFor(rows.nth(0));
    await session.hold(500);
    await setAddress(session, rows.nth(0), "3");
    await session.hold(500);

    await session.click(add, { text: true });
    await session.waitFor(rows.nth(1));
    await session.hold(800);

    await session.click(form.locator("button:has(svg.pluto-icon--play)").first());
    session.endZoom();
    await session.waitFor(form.getByText("Task started successfully").first());
    await session.hold(3000);
  } finally {
    await driver.stop();
  }
};

/** setAddress replaces the address of a channel. */
const setAddress = async (
  session: capture.CaptureSession,
  row: Locator,
  address: string,
): Promise<void> => {
  await session.click(row.locator("input").first());
  await session.press("ControlOrMeta+a");
  await session.type(address);
  await session.press("Enter");
};
