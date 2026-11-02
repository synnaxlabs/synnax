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

/** The facility's other projects: test rigs, a campaign, and its operator station. */
const PROJECTS = [
  "Control Room",
  "Hotfire Campaign",
  "Igniter Bench",
  "Propellant Farm",
  "Turbopump Rig",
  "Vertical Test Stand",
];

/** Width of the projects toolbar, wide enough for every name. */
const TOOLBAR_WIDTH = 210;

/** Each tab moved, the tab whose leaf it splits, and the edge of that leaf it takes. */
const MOVES = [
  ["Operator", "Fuel Pressures", "top"],
  ["Sequence Logs", "Fuel Pressures", "left"],
  ["OX Pressures", "Thermocouples", "top"],
] as const;

/**
 * arrange lays out the Operations panel in two columns: the plot over the table, and
 * the schematic over the log. The schematic cannot shrink to fit beside the projects
 * toolbar, so it takes the right column and runs past the edge of the hero.
 */
const arrange = async (client: ReturnType<typeof fixtures.connect>): Promise<void> => {
  const layout = await operationsLayout(client);
  for (const [name, beside, location] of MOVES)
    await layout.move(name, beside, location);
  const parent = async (name: string): Promise<number> =>
    (await layout.path(name)) >> 1;
  await layout.resize(panel.ROOT_NODE_KEY, 0.34);
  await layout.resize(await parent("OX Pressures"), 0.38);
  await layout.resize(await parent("Operator"), 0.58);
  await layout.resize(await parent("Sequence Logs"), 0.61);
};

/**
 * Still for the hero of docs page `reference/visualize/projects`: the projects toolbar
 * lists the projects of a test facility with TPC open, beside the TPC Operations panel
 * mid-coldflow. The panel bar across the top shows the other panels of the project.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await client.projects.create(PROJECTS.map((name) => ({ name })));
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

    await capture.openToolbar(session, "project");
    const toolbar = page.locator(".console-nav__drawer.lyra--location-left").first();
    // The handle listens for pointer events, which capture.resizeToolbar does not send.
    const handle = toolbar.locator(".lyra-resize__handle").first();
    const from = await handle.boundingBox();
    const width = (await toolbar.boundingBox())?.width;
    if (from == null || width == null)
      throw new Error("the projects toolbar is hidden");
    const x = from.x + from.width / 2;
    const y = from.y + from.height / 2;
    await session.drag({ x, y }, { x: x + TOOLBAR_WIDTH - width, y }, { zoom: false });
    await fitOperator(session);
    const tpc = toolbar.getByText(world.project, { exact: true }).first();
    await session.waitFor(tpc);
    await session.doubleClick(tpc, { text: true, zoom: false });

    // Partway through the coldflow, the OX tank pressure swings with its regulation.
    await session.settle(3000);
    // From just before pressurization, so both range labels fit.
    await zoomPressures(session, 0.35);

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
