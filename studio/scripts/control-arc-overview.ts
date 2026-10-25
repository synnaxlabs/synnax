// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { AUTOMATION, channels, createTankIndex } from "./common/press-sequence";

const PROGRAM = `import time

sequence press {
    stage pressurize {
        1 -> press_vlv_cmd
        tank_pt > 300 => next
    }
    0 -> press_vlv_cmd
    time.wait{duration=10s}
    stage vent {
        1 -> vent_vlv_cmd
        tank_pt < 5 => next
    }
    0 -> vent_vlv_cmd
}

start_cmd == 1 => press
`;

/**
 * Docs `control/arc/overview`: a still of a running text-mode Arc automation open in
 * the editor, with its Driver selected. The docs use the last frame.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  try {
    const time = await createTankIndex(client);
    await client.channels.create(channels(time.key), { retrieveIfNameExists: true });
  } finally {
    await client.close();
  }
  const automation = await fixtures.runArc({ name: AUTOMATION, text: PROGRAM });
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);

    await capture.openToolbar(session, "arc");
    const item = page
      .locator(".console-arc-toolbar")
      .getByText(AUTOMATION, { exact: true })
      .first();
    await session.waitFor(item);
    await session.doubleClick(item, { zoom: false, text: true });
    await session.waitFor(page.locator(".pluto-editor").first());
    await capture.closeToolbar(session);
    const controls = page.locator(".console-arc-editor__controls").first();
    await session.waitFor(controls.getByText("Task started successfully").first());
    await session.waitFor(controls.getByText("Node 1", { exact: true }).first());
    await session.moveTo({ x: 1400, y: 820 });
    await session.settle(1500);

    session.startRecording();
    // Long enough for the idle cursor to fade out of the last frame.
    await session.hold(3000);
  } finally {
    await automation.stop();
  }
};
