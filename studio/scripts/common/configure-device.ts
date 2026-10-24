// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { DRIVER } from "./task";

export interface ConfigureDeviceShot {
  /** An unconfigured device on the simulated Driver. */
  device: fixtures.SimulatedDevice;
  /** The name to give the device. */
  name: string;
  /** The identifier to give the device. */
  identifier: string;
}

/**
 * configureDeviceScript returns a script that right-clicks an unconfigured device in
 * the devices toolbar, selects Configure, enters a name and an identifier, and saves.
 * The device is simulated, and configuring it needs no Driver.
 */
export const configureDeviceScript =
  ({ device, name, identifier }: ConfigureDeviceShot) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const driver = await fixtures.simulatedDriver({ devices: [device] });
    try {
      const { page } = session;
      await capture.login(session);
      await capture.clearPanel(session);
      await capture.openToolbar(session, "device");
      await capture.resizeToolbar(session, 400);
      const rack = capture.treeItem(page, "rack:", DRIVER);
      await session.waitFor(rack);
      await session.moveTo({ x: 800, y: 450 });

      session.startRecording();
      await session.hold(1500);

      await session.zoom(rack, 1.6);
      await session.click(rack, { text: true });
      const item = capture.treeItem(page, "device:", device.name);
      await session.waitFor(item);
      await session.hold(600);
      await session.rightClick(item, { text: true });
      const menu = page.locator(".pluto-menu-context").first();
      await session.waitFor(menu);
      await session.hold(500);
      await session.click(menu.getByText("Configure", { exact: true }).first(), {
        text: true,
      });

      const modal = page.locator(".console-configure").first();
      const nameInput = modal.locator("input").first();
      await session.waitFor(nameInput);
      await session.zoom(modal);
      await session.hold(600);
      await session.press("ControlOrMeta+a");
      await session.type(name);
      await session.hold(500);
      await capture.clickButton(session, "Next");

      await session.waitFor(modal.getByText("pick a short identifier"));
      await session.hold(600);
      await session.type(identifier);
      await session.hold(600);
      await capture.clickButton(session, "Save");
      session.endZoom();
      await session.waitForHidden(modal);
      await session.waitFor(capture.treeItem(page, "device:", name));
      await session.hold(3000);
    } finally {
      await driver.stop();
    }
  };
