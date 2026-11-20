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

export { desktop, driver, integrations } from "./common/blog-labjack";

/**
 * How long the shot records after the reading starts to rise, in wall time, so the
 * plot's 30 s window holds the steady reading and the climb.
 */
const CLIMB_MS = 22_000;

/**
 * Blog `labjack-software` step 5 (Plot it live): with a person's hand warming the
 * thermocouple, select Plot channels on the started read task and watch the line climb.
 * Needs a real T7 on USB, and LABJACK_WARM=1 with a person at the T7.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  try {
    const { page } = session;
    await openProject(session);
    const form = await openTask(session);
    await readThermocouple(session, form);
    await session.moveTo(capture.PARK);
    const plot = page.locator(".pluto-line-plot").first();
    const thermocouple = await watch(client, session);
    try {
      session.startRecording();
      await session.hold(1000);

      await session.click(plotButton(form), { zoom: false });
      await session.waitFor(plot);
      await session.moveTo(capture.PARK);
      await session.hold(1500);
      thermocouple.ask();
      // Live data moves in wall time, so the shot waits in wall time. A recorded tick
      // takes longer than a frame, so the video plays the climb faster than real time.
      await thermocouple.rise(() => session.hold(100));
      const onset = Date.now();
      do await session.hold(100);
      while (Date.now() < onset + CLIMB_MS);
    } finally {
      await thermocouple.close();
    }
    letGo();
  } finally {
    await client.close();
  }
};
