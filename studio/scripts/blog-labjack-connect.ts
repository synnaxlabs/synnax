// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { configure, findT7, openProject, waitWallGone } from "./common/blog-labjack";

export { desktop, driver, integrations } from "./common/blog-labjack";

/**
 * Blog `labjack-software` step 3 (Connect the LabJack): open the connect dialog from
 * the devices toolbar, select LabJack hardware, configure the T7 that the Driver found,
 * and save it with a read task. Needs a real T7 on USB.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await openProject(session);
  await findT7(session);
  await capture.closeToolbar(session);
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
  const entry = modal
    .getByRole("button")
    .filter({ hasText: "LabJack hardware" })
    .first();
  await session.waitFor(entry);
  await session.zoom(modal);
  await session.hold(1500);
  await session.click(entry, { text: true, zoom: false });

  const found = modal.getByRole("button", { name: "Configure", exact: true });
  await session.waitFor(found);
  await session.zoom(modal);
  await session.hold(1500);
  await session.click(found, { zoom: false });
  const form = await configure(session, page.locator(".console-configure").first());
  session.endZoom();
  await session.moveTo({ x: 1000, y: 500 });
  await session.hold(600);
  // The T7 reads disconnected until the Driver's next scan reports it.
  await waitWallGone(session, form.getByText("Disconnected", { exact: true }), 30_000);
  await session.hold(3000);
};
