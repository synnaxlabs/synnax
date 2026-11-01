// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

export const STAND = "Test stand";
export const NEW_PANEL = "New panel";

/** pill returns the panel selector's pill for `name`. */
export const pill = (session: capture.CaptureSession, name: string): Locator =>
  session.page
    .locator(".console-panel-selector .lyra-tabs__tab")
    .filter({ hasText: name })
    .first();

/** selectedPill returns the pill for `name` when its panel is the visible one. */
export const selectedPill = (session: capture.CaptureSession, name: string): Locator =>
  session.page
    .locator(".console-panel-selector .lyra-tabs__tab[aria-selected='true']")
    .filter({ hasText: name })
    .first();

/** addPanel returns the add button at the right of the panel selector. */
export const addPanel = (session: capture.CaptureSession): Locator =>
  session.page.locator(".console-panel-selector > .lyra-btn").first();

/** mosaicTab returns the visible panel's tab named `name`. */
export const mosaicTab = (session: capture.CaptureSession, name: string): Locator =>
  session.page
    .locator(".console-mosaic .lyra-tabs__tab:visible")
    .filter({ hasText: name })
    .first();

/** menuItem returns a context menu item by a prefix of its text. */
const menuItem = (session: capture.CaptureSession, prefix: string): Locator =>
  // Prefix match: the item's text carries its shortcut indicator too.
  session.page
    .locator(".lyra-menu-context")
    .first()
    .getByText(new RegExp(`^${prefix}`))
    .first();

/** zoomSelector zooms on the panel selector, which the camera pins to the top edge. */
export const zoomSelector = async (session: capture.CaptureSession): Promise<void> =>
  await capture.zoomCenter(
    session,
    session.page.locator(".console-panel-selector"),
    1.5,
  );

/** renamePanel renames a panel off camera through its context menu. */
export const renamePanel = async (
  session: capture.CaptureSession,
  from: string,
  to: string,
): Promise<void> => {
  const { page } = session;
  await pill(session, from).click({ button: "right", timeout: 5000 });
  await session.waitFor(page.locator(".lyra-menu-context").first());
  await menuItem(session, "Rename").click({ timeout: 5000 });
  const editable = page.locator(".lyra-text--editable[contenteditable='true']").first();
  await session.waitFor(editable);
  await page.keyboard.press("ControlOrMeta+a");
  await page.keyboard.type(to);
  await page.keyboard.press("Enter");
  await session.waitFor(pill(session, to));
};

/** createPanel adds an empty panel off camera and names it. */
export const createPanel = async (
  session: capture.CaptureSession,
  name: string,
): Promise<void> => {
  await addPanel(session).click({ timeout: 5000 });
  await session.waitFor(selectedPill(session, NEW_PANEL));
  if (name !== NEW_PANEL) await renamePanel(session, NEW_PANEL, name);
};

/** selectPanel switches to a panel off camera. */
export const selectPanel = async (
  session: capture.CaptureSession,
  name: string,
): Promise<void> => {
  await pill(session, name).click({ timeout: 5000 });
  await session.waitFor(selectedPill(session, name));
};

export interface Stand {
  /** Names of the line plot and table tabs. */
  plot: string;
  table: string;
  /** Stops the telemetry stream. */
  stop: () => Promise<void>;
}

/**
 * createStand logs in and fills the first panel, named `name`, with a line plot of live
 * sine telemetry and a table, as two tabs in one pane with the plot visible. Call stop
 * when the shot ends.
 */
export const createStand = async (
  session: capture.CaptureSession,
  name: string = STAND,
): Promise<Stand> => {
  const fixture = await fixtures.sineTelemetry();
  let ready = false;
  try {
    const { page } = session;
    await capture.login(session);
    if (name !== NEW_PANEL) await renamePanel(session, NEW_PANEL, name);

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideBottomToolbar(session);
    const tabs = page.locator(".console-mosaic .lyra-tabs__tab");
    const plot = (await tabs.first().innerText()).trim();
    await capture.clickPanelCreate(session);
    await capture.createComponent(session, "Table");
    await session.waitFor(page.locator(".console-table").first());
    await capture.hideBottomToolbar(session);
    const table = (await tabs.nth(1).innerText()).trim();
    // An empty table is a dull first frame, so the plot is the visible tab.
    await mosaicTab(session, plot).click({ timeout: 5000 });
    await capture.hideBottomToolbar(session);
    await session.settleWall(10000);
    await session.settle(500);
    ready = true;
    return { plot, table, stop: fixture.stop };
  } finally {
    if (!ready) await fixture.stop();
  }
};

/**
 * splitRight drags a tab to the right third of its pane, which splits the pane with the
 * tab on the right.
 */
export const splitRight = async (
  session: capture.CaptureSession,
  tab: string,
): Promise<void> => {
  const leaves = session.page.locator(".console-mosaic .pluto-mosaic__leaf");
  const leaf = await leaves.first().boundingBox();
  if (leaf == null) throw new Error("the panel's pane is not on screen");
  await session.drag(
    mosaicTab(session, tab),
    { x: leaf.x + leaf.width * 0.85, y: leaf.y + leaf.height / 2 },
    { zoom: false },
  );
  await session.waitFor(leaves.nth(1));
};

/** isolateTab closes every tab of the visible panel except `name`. */
export const isolateTab = async (
  session: capture.CaptureSession,
  name: string,
): Promise<void> => {
  const others = session.page
    .locator(".console-mosaic .lyra-tabs__tab:visible")
    .filter({ hasNotText: name });
  while ((await others.count()) > 0) {
    const tab = others.first();
    await tab.hover();
    await session.click(tab.locator(".lyra-tabs__close").first(), { zoom: false });
    await session.settle(300);
  }
};
