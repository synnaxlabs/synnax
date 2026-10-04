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

/**
 * Docs `reference/hardware/modbus/read-task` (Create a Modbus read task): right-click
 * a Modbus server in the devices toolbar and create a read task, add a float32 holding
 * register at address 0 and a coil at address 3, then start the task. The server is
 * simulated, so the task reads no values.
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
    const rack = capture.treeItem(page, "rack:", "Node 1 Embedded Driver");
    await session.waitFor(rack);
    await session.click(rack, { text: true });
    const plc = capture.treeItem(page, "device:", MODBUS_PLC.name);
    await session.waitFor(plc);
    await session.hold(600);
    await session.rightClick(plc, { text: true });
    const menu = page.locator(".pluto-menu-context").first();
    await session.waitFor(menu);
    await session.hold(500);
    await session.click(menu.getByText("Create read task", { exact: true }).first(), {
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
    await choose(session, rows.nth(0), "Coil", "Holding register");
    await choose(session, rows.nth(0), "uint8", "float32");
    await session.hold(500);

    await session.click(add, { text: true });
    await session.waitFor(rows.nth(1));
    await session.hold(500);
    await choose(session, rows.nth(1), "Holding register", "Coil");
    await setAddress(session, rows.nth(1), "3");
    await session.hold(800);

    await session.click(form.locator("button:has(svg.pluto-icon--play)").first());
    session.endZoom();
    await session.waitFor(form.getByText("Task started successfully").first());
    await session.hold(3000);
  } finally {
    await driver.stop();
  }
};

/** choose opens the select in a channel that shows `current`, searches, and picks `next`. */
const choose = async (
  session: capture.CaptureSession,
  row: Locator,
  current: string,
  next: string,
): Promise<void> => {
  await session.click(
    row.locator(".pluto-dialog__trigger").filter({ hasText: current }).first(),
    { text: true },
  );
  await session.hold(300);
  await session.type(next);
  const item = session.page
    .locator(".pluto-list__item:not(.pluto-tree__item)")
    .filter({ has: session.page.getByText(next, { exact: true }) })
    .first();
  await session.waitFor(item);
  await session.hold(500);
  await session.click(item.getByText(next, { exact: true }).first(), { text: true });
  await session.hold(400);
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
