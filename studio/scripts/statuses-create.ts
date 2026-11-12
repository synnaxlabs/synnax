// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { choose, fieldItem } from "./common/task";

export const driver = true;
export const integrations = ["modbus"];

/**
 * Docs `reference/alarms/statuses` (Set a status): click the add button in the statuses
 * toolbar, enter a name, select a variant, enter a message, and create the status. The
 * status explorer then lists it.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.openToolbar(session, "status");
  await session.hold(800);
  await capture.clickToolbarCreate(session);
  const modal = page.locator(capture.MODAL).first();
  const name = modal.getByPlaceholder("Status name").first();
  await session.waitFor(name);
  await session.zoom(modal);
  await session.hold(600);
  await session.click(name, { text: true });
  await session.type("Tank pressure");
  await session.hold(400);
  await choose(session, modal, "Variant", "Warning");
  await session.click(fieldItem(modal, "Message").locator("input").first());
  await session.type("Above 600 psi");
  await session.hold(600);
  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(modal);
  await session.hold(800);
  // A new status is no favorite, so the toolbar stays empty and the explorer lists it.
  await session.click(page.getByText("Open status explorer", { exact: true }).first(), {
    text: true,
    zoom: false,
  });
  await session.waitFor(page.getByText("Above 600 psi").first());
  await session.moveTo({ x: 900, y: 600 });
  await session.hold(3000);
};
