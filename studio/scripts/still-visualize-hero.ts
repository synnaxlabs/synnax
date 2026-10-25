// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

import {
  fitOperator,
  openOperator,
  operationsLayout,
  startColdflow,
  zoomPressures,
} from "./common/coldflow";

export const viewport = { width: 1280, height: 800 };

/** Each tab moved, the tab whose leaf it splits, and the edge of that leaf it takes. */
const MOVES = [
  ["OX Pressures", "Operator", "right"],
  ["Sequence Logs", "Operator", "bottom"],
  ["Thermocouples", "OX Pressures", "bottom"],
] as const;

/**
 * arrange lays out the Operations panel so the schematic, the log under it, the plot,
 * and the table under it fill the part of the window the hero shows. The fuel plot
 * takes the column past its right edge.
 */
const arrange = async (client: ReturnType<typeof fixtures.connect>): Promise<void> => {
  const layout = await operationsLayout(client);
  for (const [name, beside, location] of MOVES)
    await layout.move(name, beside, location);
  const left = (await layout.path("Operator")) >> 1;
  const right = (await layout.path("OX Pressures")) >> 1;
  // The fuel column starts just past the right edge of the part the hero shows.
  await layout.resize(panel.ROOT_NODE_KEY, 0.765);
  await layout.resize(left >> 1, 0.652);
  await layout.resize(left, 0.582);
  await layout.resize(right, 0.35);
};

/**
 * Still for the hero of docs page `reference/visualize`: the TPC stand's Operations
 * panel mid-coldflow, with its schematic, log, line plot, and table live together. The
 * hero shows only the top left of the window, which the four of them fill.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await arrange(client);
    await capture.login(session, world.project);
    const leaf = (text: string): Locator =>
      page.locator(".pluto-mosaic__leaf:visible").filter({ hasText: text }).first();
    const diagram = page.locator(".pluto-diagram:visible").first();

    await openOperator(session);
    await capture.control(diagram, "edit-off").click();
    await fitOperator(session);

    await startColdflow(session, world, client);
    const table = leaf("Thermocouples").locator(".console-table").first();
    await capture.contextMenu(session, table, "Align table to top left");
    session.endZoom();
    // Released, the schematic's controls fade out instead of staying pinned.
    await capture.control(diagram, "circle").click();
    // Partway through the coldflow, the OX tank pressure swings with its regulation.
    await session.settle(4000);
    // From just before pressurization, so both range labels fit.
    await zoomPressures(session, 0.5);

    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    // Canvas content draws on wall time, which virtual ticks outrun.
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};
