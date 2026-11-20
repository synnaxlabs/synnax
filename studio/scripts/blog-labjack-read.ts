// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { openProject, openTask } from "./common/blog-labjack";
import { choose, fieldItem } from "./common/task";

export { desktop, driver, integrations } from "./common/blog-labjack";

/**
 * Blog `labjack-software` step 4 (Read the thermocouple): make the read task's AIN0
 * channel a thermocouple in Celsius, and start the task. Needs a real T7 on USB.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await openProject(session);
  const form = await openTask(session);
  await session.moveTo(capture.PARK);

  session.startRecording();
  await session.hold(1500);

  await capture.zoomCenter(session, form, 1.2);
  await choose(session, form, "Channel type", "Thermocouple");
  await fieldItem(form, "Temperature units").scrollIntoViewIfNeeded();
  await choose(session, form, "Temperature units", "Celsius");
  await session.hold(600);
  session.endZoom();
  await session.click(form.locator("button:has(svg.lyra-icon--play)").first(), {
    zoom: false,
  });
  await session.waitFor(form.getByText("Started task").first(), 1800);
  await session.moveTo(capture.PARK);
  await session.hold(3000);
};
