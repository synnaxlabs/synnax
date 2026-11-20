// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import {
  at,
  letGo,
  openProject,
  openTask,
  plotButton,
  readThermocouple,
  warmUp,
} from "./common/blog-labjack";

export { desktop, driver, integrations } from "./common/blog-labjack";

/** How long the warm-up climbs before the shot, in wall time. */
const CLIMB_MS = 25_000;

/**
 * Blog `labjack-software` step 6 (Record and export): hold Alt, drag across the warm-up
 * on the live plot, and open the selection's menu on Download region as CSV. Needs a
 * real T7 on USB, and LABJACK_WARM=1 with a person at the T7.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  try {
    const { page } = session;
    await openProject(session);
    const form = await openTask(session);
    await readThermocouple(session, form);
    await session.click(plotButton(form));
    const plot = page.locator(".pluto-line-plot").first();
    await session.waitFor(plot);
    await session.moveTo(capture.PARK);
    // Fill the plot's 30 s window with the steady reading before the climb.
    await session.settleWall(15_000);
    await warmUp(client, session);
    await session.settleWall(CLIMB_MS);
    // A recorded tick takes longer than a frame, so a live plot would scroll the climb
    // away during the shot.
    await session.click(capture.control(page, "pause"), { zoom: false });
    await session.moveTo(capture.PARK);
    letGo();

    session.startRecording();
    await session.hold(1500);

    await page.keyboard.down("Alt");
    await session.drag(await at(plot, 0.3, 0.12), await at(plot, 0.95, 0.88), {
      zoom: false,
    });
    await page.keyboard.up("Alt");
    await session.hold(400);
    await session.rightClick(await at(plot, 0.62, 0.5));
    const menu = page.locator(".lyra-menu-context").first();
    await session.waitFor(menu);
    await session.hold(600);
    await session.moveTo(menu.getByText("Download region as CSV").first());
    await session.hold(3000);
  } finally {
    await client.close();
  }
};
