// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

import { DRIVER, fieldItem, rackItem } from "./task";

/** The EL3102 as the Driver finds it, with the modes that its ESI file lists. */
const SLAVE: fixtures.SimulatedDevice = {
  ...fixtures.EL3102,
  properties: {
    ...fixtures.EL3102.properties,
    op_modes: [
      {
        name: "SM-Synchron",
        assign_activate: 0,
        sync0_cycle: 0,
        sync0_factor: 0,
        sync0_shift: 0,
        sync1_factor: 0,
        sync1_shift: 0,
      },
      {
        name: "DC-Synchron",
        assign_activate: 0x700,
        sync0_cycle: 0,
        sync0_factor: 1,
        sync0_shift: 0,
        sync1_factor: 1,
        sync1_shift: 0,
      },
    ],
  },
};

/**
 * configureScript returns a script that opens the Configure dialog of the simulated
 * slave, fills its name and identifier, runs `shot` in the dialog, and saves.
 */
export const configureScript =
  (shot: (session: capture.CaptureSession, modal: Locator) => Promise<void>) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const driver = await fixtures.simulatedDriver({ devices: [SLAVE] });
    try {
      const { page } = session;
      await capture.login(session);
      await capture.clearPanel(session);
      await capture.openToolbar(session, "device");
      await capture.resizeToolbar(session, 400);
      const rack = rackItem(page, DRIVER);
      await session.waitFor(rack);
      await session.click(rack, { text: true });
      const item = capture.treeItem(page, "device:", SLAVE.name);
      await session.waitFor(item);
      await session.moveTo({ x: 800, y: 450 });

      session.startRecording();
      await session.hold(1500);

      await capture.contextMenu(session, item, "Configure");
      const modal = page.locator(".console-ethercat-configure__frame").first();
      const name = modal.locator("input").first();
      await session.waitFor(name);
      await session.zoom(modal);
      await session.hold(600);
      await session.click(name, { text: true });
      await session.press("ControlOrMeta+a");
      await session.type("Tank pressures");
      await session.hold(300);
      await session.click(fieldItem(modal, "Identifier").locator("input").first(), {
        text: true,
      });
      await session.type("ai1");
      await session.hold(500);

      await shot(session, modal);

      await capture.clickButton(session, "Save");
      session.endZoom();
      await session.waitForHidden(modal);
      await session.waitFor(capture.treeItem(page, "device:", "Tank pressures"));
      await session.hold(3000);
    } finally {
      await driver.stop();
    }
  };
