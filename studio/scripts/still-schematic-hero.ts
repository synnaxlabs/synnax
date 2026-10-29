// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { openOperator, startColdflow } from "./common/coldflow";

export const viewport = { width: 1280, height: 800 };

/** The part of the window the hero shows, in CSS pixels from its top left corner. */
const VISIBLE = { right: 981, bottom: 593 };

/**
 * Still for the hero of docs page `reference/visualize/schematics`: the TPC stand's
 * Operator schematic alone in the Operations panel while the OX main valve is open in a
 * coldflow, framed inside the part of the window the hero shows.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await capture.login(session, world.project);
    const diagram = page.locator(".pluto-diagram:visible").first();
    const drawer = page
      .locator(".console-nav__drawer.lyra--visible.lyra--location-bottom")
      .first();

    await openOperator(session);

    await startColdflow(session, world, client);
    // The OX main valve closes seconds after it opens, so the rest spends little
    // virtual time.
    const others = page
      .locator(".console-mosaic .lyra-tabs__tab:visible")
      .filter({ hasNotText: "Operator" });
    while ((await others.count()) > 0) {
      const tab = others.first();
      await tab.hover();
      await tab.locator(".lyra-tabs__close").first().click();
      await session.settle(100);
    }
    // Released, the schematic's controls fade out instead of staying pinned.
    const held = diagram.locator(
      '.console-controls button[aria-pressed="true"]:has(.lyra-icon--circle)',
    );
    if ((await held.count()) > 0) await held.first().click();

    // With the bottom toolbar open, the panel is as tall as the part the hero shows, so
    // the fit sizes the schematic to it. Closing the toolbar keeps that viewport.
    const toggle = page.locator(".console-main-nav__item").last();
    await toggle.click();
    await session.waitFor(drawer);
    await capture.fitView(session);
    const box = await diagram.boundingBox();
    if (box == null) throw new Error("the schematic is not on screen");
    await toggle.click();
    await session.waitForHidden(drawer);

    // The fit centers the schematic in the panel, so a pan moves it to the center of
    // the part the hero shows.
    const from = { x: box.x + box.width - 60, y: box.y + box.height / 2 };
    const to = {
      x: from.x + (VISIBLE.right - (box.x + box.width)) / 2,
      y: from.y + (VISIBLE.bottom - (box.y + box.height)) / 2,
    };
    await page.keyboard.down("Shift");
    await session.drag(from, to, { zoom: false });
    await page.keyboard.up("Shift");

    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    await session.settle(600);
    // The OX tank pressure rises only while OX TPC 1 is open.
    let last = world.read("ox_pt_1");
    await session.waitFor(() => {
      const now = world.read("ox_pt_1");
      const rising = now > last + 0.2;
      last = now;
      return rising && now > 20 && now < 25;
    });
    // The stand holds still while the schematic catches up on wall time.
    await session.settleWall(1500);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};
