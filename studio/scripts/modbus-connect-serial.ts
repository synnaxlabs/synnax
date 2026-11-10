// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { INTEGRATIONS } from "./common/server";
import { choose, DRIVER, fieldItem } from "./common/task";

export const driver = true;
export const integrations = INTEGRATIONS;

/**
 * Docs `reference/hardware/modbus/connect-server` (Connect a Modbus server): connect a
 * server on a serial port by its port and line settings. The shot ends before Connect,
 * because the computer that records it has no serial device.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.openToolbar(session, "device");
  await session.hold(600);
  await capture.clickToolbarCreate(session);
  const modal = page.locator(capture.MODAL).first();
  const entry = modal.getByRole("button").filter({ hasText: "Modbus server" }).first();
  await session.waitFor(entry);
  await session.zoom(modal);
  await session.hold(1200);
  await session.click(entry, { text: true, zoom: false });

  const name = modal.getByPlaceholder("Modbus server").first();
  await session.waitFor(name);
  await session.hold(800);
  await session.click(name, { text: true });
  await session.press("ControlOrMeta+a");
  await session.type("Pump skid RTU");
  await session.hold(400);

  await session.click(
    fieldItem(modal, "Transport").getByText("Serial", { exact: true }),
    {
      text: true,
    },
  );
  const port = modal.getByPlaceholder("/dev/ttyUSB0").first();
  await session.waitFor(port);
  await session.hold(600);
  await session.click(port);
  await session.type("/dev/ttyUSB0");
  await session.hold(400);
  await choose(session, modal, "Baud rate", "19200");
  await choose(session, modal, "Parity", "Even");
  await session.waitFor(fieldItem(modal, "Connect from").getByText(DRIVER));
  await session.hold(3000);
  session.endZoom();
  await session.hold(1500);
};
