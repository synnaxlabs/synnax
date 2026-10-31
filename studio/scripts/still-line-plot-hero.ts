// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { startColdflow } from "./common/coldflow";
import { isolateTab } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

/**
 * Still for the hero of docs page `reference/visualize/line-plots`: the TPC stand's OX
 * Pressures plot alone in the Operations panel through a whole coldflow, with its
 * toolbar open. The still records the box of the toolbar, which the hero lifts out of
 * the window.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await capture.login(session, world.project);
    await startColdflow(session, world, client);
    const plot = page
      .locator(".pluto-mosaic__leaf:visible")
      .filter({ hasText: "OX Pressures" })
      .first();
    await session.click(plot.getByText("OX Pressures", { exact: true }).first());
    await capture.addChannels(session, "Y2", ["press_pt_1", "press_pt_2"], {
      search: "press",
    });
    await session.waitFor(() => world.logged.includes("OX Vent open"), 3600);
    // Held as the tank vents, the plot shows the whole coldflow.
    await session.settle(2500);
    await session.click(capture.control(plot, "pause"), { zoom: false });

    await isolateTab(session, "OX Pressures");
    await session.click(plot.getByText("OX Pressures", { exact: true }).first());
    await capture.showVisualizationToolbar(session);
    const toolbar = page.locator(".console-nav__drawer.lyra--location-bottom").first();
    await session.waitFor(toolbar);
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    // The plot redraws at its new size on wall time, which virtual ticks outrun.
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window", { toolbar });
  } finally {
    await client.close();
    await world.stop();
  }
};
