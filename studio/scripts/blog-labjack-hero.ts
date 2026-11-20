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
  letGo,
  openProject,
  openTask,
  plotButton,
  readThermocouple,
  watch,
} from "./common/blog-labjack";
import { NEW_PANEL, renamePanel, splitRight } from "./common/panels";

export { desktop, driver, integrations } from "./common/blog-labjack";

export const viewport = { width: 1280, height: 800 };

const PANEL = "Lab T7";
const TASK = "LabJack read task";
/**
 * How long the plot climbs after the reading starts to rise, in wall time, so its 30 s
 * window shows the steady reading and then the climb.
 */
const CLIMB_MS = 20_000;

/**
 * The hero still of the blog post `labjack-software`, from a real LabJack T7 on USB
 * with a type K thermocouple on `AIN0` and `GND`: a live plot of the thermocouple as a
 * hand warms it, beside the read task that streams it. Fails when no T7 is on USB, or
 * when the reading does not start to rise within three minutes.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  try {
    const { page } = session;
    await openProject(session);
    await renamePanel(session, NEW_PANEL, PANEL);
    const form = await openTask(session);
    await readThermocouple(session, form);
    await session.click(plotButton(form), { zoom: false });
    const plot = page.locator(".pluto-line-plot").first();
    await session.waitFor(plot);
    await splitRight(session, TASK);
    await session.moveTo(capture.PARK);

    const thermocouple = await watch(client, session);
    try {
      thermocouple.ask();
      await thermocouple.rise(() => session.settleWall(250));
      await session.settleWall(CLIMB_MS);
    } finally {
      await thermocouple.close();
    }
    await session.settle(600);
    await session.still("window");
    letGo();
  } finally {
    await client.close();
  }
};
