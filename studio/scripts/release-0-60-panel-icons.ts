// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, type Synnax } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { startColdflow, zoomPressures } from "./common/coldflow";

export const viewport = { width: 1280, height: 800 };

/** The icon and color of each panel of the TPC project. */
const STYLES = {
  Automations: { icon: "arc", color: "#8075FF" },
  Fuel: { icon: "flame", color: "#F18F01" },
  Hardware: { icon: "rack", color: "#7AC74F" },
  Operations: { icon: "dashboard", color: "#DC136C" },
  Oxidizer: { icon: "snowflake", color: "#20A4F3" },
  Pressurant: { icon: "gas-cylinder", color: "#FFC43D" },
  Review: { icon: "checklist", color: "#FE5F55" },
} as const;

/** The panel whose icon picker the still shows. */
const PICKED = "Fuel";

const style = async (client: Synnax): Promise<void> => {
  for (const [name, { icon, color }] of Object.entries(STYLES)) {
    const found = (await client.panels.retrieve({ searchTerm: name })).find(
      (p) => p.name === name,
    );
    if (found == null) throw new Error(`no panel named "${name}"`);
    const actions = [panel.setIcon({ icon }), panel.setColor({ color })];
    if (!(await client.panels.dispatch(found.key, actions)))
      throw new Error(`the ${name} panel is not cached`);
  }
};

/**
 * Still for the panel icons section of the v0.60 release notes: the TPC stand
 * mid-coldflow, with an icon and a color on each panel of the panel selector, and the
 * icon picker of the Fuel panel open, its grid tinted with the panel's orange.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await style(client);
    await capture.login(session, world.project);
    const diagram = page.locator(".pluto-diagram:visible").first();

    await startColdflow(session, world, client);
    // Released, the schematic's controls fade out instead of staying pinned.
    await capture.control(diagram, "circle").click();
    await session.settle(3000);
    await zoomPressures(session, 0.35);

    const pill = page
      .locator(".console-panel-selector .lyra-tabs__tab")
      .filter({ has: page.getByText(PICKED, { exact: true }) })
      .first();
    await session.rightClick(pill, { text: true });
    const menu = page.locator(".lyra-menu-context").first();
    await session.waitFor(menu);
    await session.click(menu.getByText("Change icon", { exact: true }).first());
    await session.waitFor(page.locator(".console-panel-icon-picker").first());

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
