// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import {
  fitOperator,
  openOperator,
  operationsLayout,
  startColdflow,
  zoomPressures,
} from "./common/coldflow";

export const viewport = { width: 1280, height: 800 };

/** Each TPC panel's icon, and its color when it does not use the theme's. */
const ICONS: Record<string, { icon: string; color?: string }> = {
  Automations: { icon: "arc", color: "#A855F7" },
  Fuel: { icon: "flame", color: "#FF9B42" },
  Hardware: { icon: "rack" },
  Operations: { icon: "rocket", color: "#3774D0" },
  Oxidizer: { icon: "snowflake", color: "#0FA3A3" },
  Pressurant: { icon: "gas-cylinder", color: "#F4CA25" },
  Review: { icon: "line-plot", color: "#50C878" },
};

/** Each tab moved, the tab whose leaf it splits, and the edge of that leaf it takes. */
const MOVES = [
  ["Operator", "Fuel Pressures", "top"],
  ["Sequence Logs", "Fuel Pressures", "left"],
  ["OX Pressures", "Thermocouples", "top"],
] as const;

/**
 * Still for the hero of the v0.60 release notes: the TPC stand's Operations panel
 * partway through a coldflow, with the plot over the table on the left and the
 * schematic over the log on the right. Each panel in the selector carries its own icon
 * and color.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    const panels = await client.ontology.retrieve({ types: ["panel"] });
    for (const { id, name } of panels) {
      const look = ICONS[name];
      if (look == null) continue;
      // Dispatch applies to a cached panel only, and a retrieve caches it.
      await client.panels.retrieve(id.key);
      const actions = [panel.setIcon({ icon: look.icon })];
      if (look.color != null) actions.push(panel.setColor({ color: look.color }));
      const applied = await client.panels.dispatch(id.key, actions);
      if (!applied) throw new Error(`the ${name} panel is not cached`);
    }
    const layout = await operationsLayout(client);
    for (const [name, beside, location] of MOVES)
      await layout.move(name, beside, location);
    const parent = async (name: string): Promise<number> =>
      (await layout.path(name)) >> 1;
    await layout.resize(panel.ROOT_NODE_KEY, 0.42);
    await layout.resize(await parent("OX Pressures"), 0.56);
    await layout.resize(await parent("Operator"), 0.66);

    await capture.login(session, world.project);
    const diagram = page.locator(".pluto-diagram:visible").first();

    await openOperator(session);
    await capture.control(diagram, "edit-off").click();
    await fitOperator(session);

    await startColdflow(session, world, client);
    // Released, the schematic's controls fade out instead of staying pinned.
    await capture.control(diagram, "circle").click();
    await fitOperator(session);

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
