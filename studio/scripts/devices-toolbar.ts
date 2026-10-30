// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { DRIVER } from "./common/task";

/**
 * Docs `reference/hardware/devices` (Devices toolbar): open the devices toolbar, expand
 * the embedded Driver to show its devices, then right-click a device to show its menu.
 * The devices are simulated.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const driver = await fixtures.simulatedDriver({
    devices: [fixtures.NI_USB_6343, fixtures.LABJACK_T7, fixtures.MODBUS_PLC],
  });
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await capture.openToolbar(session, "device");
    await capture.resizeToolbar(session, 400);
    await capture.openToolbar(session, "task");
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1500);

    await capture.openToolbar(session, "device");
    await session.hold(800);
    const rack = capture.treeItem(page, "rack:", DRIVER);
    await session.waitFor(rack);
    await session.click(rack, { text: true });
    const ni = capture.treeItem(page, "device:", fixtures.NI_USB_6343.name);
    await session.waitFor(ni);
    await session.hold(600);
    await session.moveTo(ni);
    await session.hold(1500);

    await session.rightClick(ni, { text: true, zoom: false });
    const menu = page.locator(".lyra-menu-context").first();
    await session.waitFor(menu);
    await session.zoom([ni, menu]);
    await session.hold(600);
    for (const label of ["Rename", "Configure", "Create analog read task"]) {
      await session.moveTo(menu.getByText(label, { exact: true }).first());
      await session.hold(500);
    }
    await session.press("Escape");
    session.endZoom();
    await session.waitForHidden(menu);
    await session.hold(2500);
  } finally {
    await driver.stop();
  }
};
