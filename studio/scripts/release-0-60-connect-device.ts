// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

export const viewport = { width: 1280, height: 800 };

export const driver = true;

/**
 * The integrations of the Core's embedded Driver. LabJack stays off, since its scanner
 * would contend for the LabJack of the machine that runs the capture.
 */
export const integrations = ["opc", "modbus", "http", "ethercat", "arc"];

/**
 * Still for the Connect device section of the v0.60 release notes: the TPC stand with
 * the Devices toolbar open on its racks and the Connect device picker open over the
 * Operations panel. The embedded Driver connects servers by address, and the stand's
 * DAQ rack finds NI hardware.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  try {
    const { page } = session;
    await capture.login(session, world.project);
    await session.click(page.getByText("Operations", { exact: true }).first());
    await session.waitFor(page.locator(".pluto-diagram:visible").first());
    await session.settle(1500);

    await capture.openToolbar(session, "device");
    // Dispatched clicks leave no row in its hover state.
    const rack = capture.treeItem(page, "rack:", "Test stand DAQ");
    await session.waitFor(rack);
    await rack.dispatchEvent("click");
    await session.waitFor(capture.treeItem(page, "device:", "NI PXIe-6363"));
    await session.settle(400);

    await capture.clickToolbarCreate(session);
    const picker = page.locator(".console-device-connect").first();
    await session.waitFor(picker);
    // Each entry reads its warning from the racks, which load after the picker opens.
    await session.waitFor(picker.getByText("Find hardware the Driver detects").first());
    // On the backdrop clear of the picker, so nothing shows its hover state.
    await session.moveTo({ x: 1240, y: 760 });
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await world.stop();
  }
};
