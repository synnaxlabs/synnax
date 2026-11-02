// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { DRIVER, fieldItem } from "./task";

/**
 * configureDeviceScript returns a script that right-clicks `device`, an unconfigured
 * device, in the devices toolbar, selects Configure, enters the name and identifier of
 * `configured`, and saves. The device is simulated, and configuring it needs no Driver.
 * Throws when `configured` has no identifier.
 */
export const configureDeviceScript =
  (device: fixtures.SimulatedDevice, configured: fixtures.SimulatedDevice) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const { name } = configured;
    const { identifier } = configured.properties;
    if (typeof identifier !== "string") throw new Error(`${name} has no identifier`);
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

      await session.click(rack, { text: true });
      const item = capture.treeItem(page, "device:", device.name);
      await session.waitFor(item);
      await session.hold(600);
      await capture.contextMenu(session, item, "Configure");

      const modal = page.locator(".console-configure").first();
      const nameInput = modal.locator("input").first();
      await session.waitFor(nameInput);
      await session.zoom(modal);
      await session.hold(600);
      await session.press("ControlOrMeta+a");
      await session.type(name);
      await session.hold(500);
      await session.click(fieldItem(modal, "Identifier").locator("input").first(), {
        text: true,
      });
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
