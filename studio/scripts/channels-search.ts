// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator, type Page } from "playwright";

import { capture, fixtures } from "@/index";

/** channelItem returns the channels toolbar item named exactly `name`. */
const channelItem = (page: Page, name: string): Locator =>
  page
    .locator("div[id^='channel:']")
    .filter({ has: page.getByText(name, { exact: true }) })
    .first();

/** createGroup groups two channels under `name`, off camera. */
const createGroup = async (
  session: capture.CaptureSession,
  [first, second]: [string, string],
  name: string,
): Promise<void> => {
  const { page } = session;
  await session.waitFor(channelItem(page, second));
  await session.click(channelItem(page, first), { text: true, zoom: false });
  await page.keyboard.down("ControlOrMeta");
  await session.click(channelItem(page, second), { text: true, zoom: false });
  await page.keyboard.up("ControlOrMeta");
  await session.rightClick(channelItem(page, second), { text: true, zoom: false });
  const menu = page.locator(".lyra-menu-context").first();
  await session.waitFor(menu);
  await session.click(menu.getByText("Group selection", { exact: true }).first(), {
    text: true,
    zoom: false,
  });
  await session.waitFor(
    page.locator(".lyra-text--editable[contenteditable='true']").first(),
  );
  await session.type(name);
  await session.press("Enter");
  await session.settle(800);
};

/**
 * Docs `reference/channels/find` (Search the channels toolbar): click search in the
 * toolbar header, type part of a name, and show the matches in their groups.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const tank = await fixtures.sineTelemetry({ prefix: "tank" });
  const oxTank = await fixtures.sineTelemetry({ prefix: "ox_tank", base: 40 });
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await capture.openToolbar(session, "channel");
    await capture.resizeToolbar(session, 400);
    await createGroup(session, ["tank_pressure", "tank_temperature"], "Tank");
    await createGroup(session, ["ox_tank_pressure", "ox_tank_temperature"], "OX tank");
    await capture.closeToolbar(session);
    await session.moveTo({ x: 900, y: 450 });

    session.startRecording();
    await session.hold(1500);

    await capture.openToolbar(session, "channel");
    const drawer = page
      .locator(".console-nav__drawer.lyra--visible:not(.lyra--location-bottom)")
      .first();
    await session.waitFor(drawer);
    await session.hold(600);
    await capture.zoomCenter(session, drawer, 1.5);
    await session.hold(400);

    await session.click(drawer.locator("button:has(svg.lyra-icon--search)").first());
    await session.waitFor(drawer.getByPlaceholder("Search channels").first());
    await session.hold(400);
    await session.type("pressure");
    await session.waitFor(channelItem(page, "ox_tank_pressure"));
    await session.hold(1600);

    await session.click(drawer.locator("button:has(svg.lyra-icon--tree-view)").first());
    await session.hold(2200);
    session.endZoom();
    await session.moveTo({ x: 900, y: 450 });
    await session.hold(2500);
  } finally {
    await tank.stop();
    await oxTank.stop();
  }
};
