// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { DRIVER, fieldItem, NI_DAQ } from "./common/task";

/** Every integration in the connect dialog, so that no entry shows a warning. */
const INTEGRATIONS = [
  "opc",
  "modbus",
  "http",
  "ni",
  "labjack",
  "ethercat",
  "mqtt",
  "kafka",
];

/**
 * Docs `reference/hardware/devices` (Connect a device): open the connect dialog from
 * the devices toolbar, select NI hardware, and configure the device that the Driver
 * found. The device then shows under the Driver. The Driver and the device are
 * simulated.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const driver = await fixtures.simulatedDriver({
    devices: [fixtures.NI_USB_6343],
    integrations: INTEGRATIONS,
    scanners: ["ni_scanner"],
  });
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1500);

    const devices = page
      .locator("button.console-main-nav__item")
      .filter({ has: page.locator("svg.lyra-icon--device") })
      .first();
    await capture.zoomCenter(session, devices, 1.6);
    await session.hold(800);
    await capture.openToolbar(session, "device");
    await session.hold(1000);
    await capture.clickToolbarCreate(session);
    const modal = page.locator(capture.MODAL).first();
    const entry = modal.getByRole("button").filter({ hasText: "NI hardware" }).first();
    await session.waitFor(entry);
    await session.zoom(modal);
    await session.hold(2000);
    await session.click(entry, { text: true, zoom: false });

    const configure = modal.getByRole("button", { name: "Configure", exact: true });
    await session.waitFor(configure);
    await session.zoom(modal);
    await session.hold(1500);
    await session.click(configure, { zoom: false });

    const form = page.locator(".console-configure").first();
    const nameInput = form.locator("input").first();
    await session.waitFor(nameInput);
    await session.zoom(form);
    await session.hold(600);
    await session.press("ControlOrMeta+a");
    await session.type(NI_DAQ.name);
    await session.hold(500);
    await session.click(fieldItem(form, "Identifier").locator("input").first(), {
      text: true,
    });
    await session.type(String(NI_DAQ.properties.identifier));
    await session.hold(600);
    await capture.clickButton(session, "Save");
    await session.waitFor(modal.getByText("Configured", { exact: true }));
    await session.hold(1500);

    await session.press("Escape");
    await session.waitForHidden(modal);
    const rack = capture.treeItem(page, "rack:", DRIVER);
    await session.waitFor(rack);
    await session.zoom(rack, 1.6);
    await session.hold(600);
    await session.click(rack, { text: true });
    await session.waitFor(capture.treeItem(page, "device:", NI_DAQ.name));
    await session.hold(1500);
    session.endZoom();
    await session.moveTo({ x: 1000, y: 500 });
    await session.hold(3000);
  } finally {
    await driver.stop();
  }
};
