// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import {
  button,
  controls,
  editor,
  openAutomation,
  openSample,
  symbol,
  tab,
  watchTank,
} from "./common/sample";

export const desktop = true;

const PROGRAM = `sequence press {
    stage idle {
        0 -> sample_press_vlv_cmd
        0 -> sample_vent_vlv_cmd
        sample_start_cmd == 1 => pressurize
    }
    stage pressurize {
        1 -> sample_press_vlv_cmd
        0 -> sample_vent_vlv_cmd
    }
}

true => press
`;

/**
 * Docs `control/arc/get-started/first-stages`: paste the two-stage sequence into a new
 * automation, start it, and click Start on the Tank schematic. The press valve opens
 * and the pressure climbs.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await openSample(session);
  await openAutomation(session, { name: "My press sequence", text: "" });
  const tank = await watchTank();
  try {
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);
    await session.zoom(editor(page), 1.4);
    await session.click(editor(page));
    await session.hold(400);
    await session.paste(PROGRAM);
    await session.hold(1200);
    await session.zoom(controls(page), 1.6);
    await session.click(controls(page).locator("button:has(svg.lyra-icon--play)"));
    await session.waitFor(controls(page).getByText("Started task"));
    await session.hold(1000);
    session.endZoom();
    await session.click(tab(page, "Tank"), { zoom: false });
    await session.waitFor(symbol(page, "Start"));
    await session.hold(600);
    await session.zoom([symbol(page, "Start"), symbol(page, "Abort")], 1.6);
    await session.click(button(page, "Start"));
    await session.waitFor(() => tank.press === 1);
    session.endZoom();
    await session.moveTo(capture.PARK);
    await session.waitFor(() => tank.pressure > 300, 3600);
    await session.hold(2500);
  } finally {
    await tank.stop();
  }
};
