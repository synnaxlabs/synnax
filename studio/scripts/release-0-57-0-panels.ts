// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const STAND = "Stand";
const REVIEW = "Review";
const NEW_PANEL = "New panel";

/** pill returns the panel strip's pill for `name`. */
const pill = (session: capture.CaptureSession, name: string) =>
  session.page
    .locator(".console-panel-selector .pluto-tabs__tab")
    .filter({ hasText: name })
    .first();

/** mosaicTabs returns the panel's own tabs; the strip renders tabs too. */
const mosaicTabs = (session: capture.CaptureSession) =>
  session.page.locator(".console-mosaic .pluto-tabs__tab");

/**
 * renamePanel renames the panel named `from`. Setup only: the pill's Rename
 * item carries a shortcut indicator, so an exact text match misses it.
 */
const renamePanel = async (
  session: capture.CaptureSession,
  from: string,
  to: string,
): Promise<void> => {
  const { page } = session;
  await pill(session, from).click({ button: "right", timeout: 5000 });
  const menu = page.locator(".pluto-menu-context").first();
  await session.waitFor(menu);
  await menu
    .getByText(/^Rename/)
    .first()
    .click({ timeout: 5000 });
  const editable = page
    .locator(".pluto-text--editable[contenteditable='true']")
    .first();
  await session.waitFor(editable);
  await page.keyboard.type(to);
  await page.keyboard.press("Enter");
  await session.waitFor(pill(session, to));
};

/**
 * Docs `releases/0-57-0/panels`: a project holding a stand schematic in one
 * panel and a streaming plot in another. Switch between them from the strip,
 * then move the schematic into the plot's panel from the tab menu.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session, { username: "synnax", password: "seldon" });

    await renamePanel(session, NEW_PANEL, STAND);
    await capture.createComponent(session, "Schematic");
    await session.waitFor(page.locator(".pluto-diagram").first());
    const tank = await capture.place(session, "Tank", { x: 430, y: 420 });
    const valve = await capture.place(session, "Gate", { x: 800, y: 420 });
    const pump = await capture.place(session, "Pump", { x: 1120, y: 420 });
    await capture.deselect(session, { x: 1350, y: 750 });
    await session.drag(capture.handle(tank, "right"), capture.handle(valve, "left"), {
      zoom: false,
    });
    await session.drag(capture.handle(valve, "right"), capture.handle(pump, "left"), {
      zoom: false,
    });
    // A drop onto a handle leaves the new edge selected, which paints a box and
    // an edge control over the symbol it landed on.
    await page.keyboard.press("Escape");
    await capture.deselect(session, { x: 640, y: 700 });
    await capture.hideBottomToolbar(session);

    // A minted panel opens empty, not on the component selector.
    await page.locator(".console-panel-selector > .pluto-btn").first().click();
    await session.waitFor(page.getByText("No components open").first());
    await renamePanel(session, NEW_PANEL, REVIEW);
    await capture.clickPanelCreate(session);
    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideBottomToolbar(session);
    // The tab's name is read here, while the plot is alone in this panel; after
    // the move it shares the strip with the schematic.
    const plotTab = (await mosaicTabs(session).first().innerText()).trim();
    // Live telemetry fills on wall time, so the plot would record as an empty
    // grid if the shot started before the rolling window filled.
    await session.settleWall(24000);

    await pill(session, STAND).click();
    await session.waitFor(page.locator(".pluto-diagram").first());
    await session.settle(1000);
    await session.moveTo({ x: 756, y: 730 });

    session.startRecording();
    await session.hold(1200);

    // No zoom on either pill: the click swaps the whole panel, so the camera
    // has nothing to punch into and the receding zoom would smear the swap.
    await session.click(pill(session, REVIEW), { text: true, zoom: false });
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await session.hold(2200);

    await session.click(pill(session, STAND), { text: true, zoom: false });
    await session.waitFor(page.locator(".pluto-diagram").first());
    await session.hold(1600);

    await session.rightClick(mosaicTabs(session).first(), { zoom: false });
    const menu = page.locator(".pluto-menu-context").first();
    await session.waitFor(menu);
    await session.hold(300);
    await session.click(menu.getByText(/^Move to panel/).first(), { zoom: false });
    const picker = page.locator(".console-panel-move-picker").first();
    await session.waitFor(picker);
    await session.hold(800);
    // No zoom: the picker is a short list, and punching into it drops the panel
    // strip the shot is about out of frame.
    await session.click(picker.getByText(REVIEW, { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    await session.waitForHidden(picker);

    // The move follows the tab: the Review panel is selected and the schematic
    // is on screen inside it.
    await session.waitFor(mosaicTabs(session).nth(1));
    await session.hold(1400);

    await session.click(mosaicTabs(session).filter({ hasText: plotTab }).first(), {
      text: true,
      zoom: false,
    });
    await session.waitFor(page.locator(".pluto-line-plot").first());
    // Park off the panel: the tab keeps its close button under the pointer, and
    // anywhere on the plot raises the crosshair and its value readout.
    await session.moveTo({ x: 900, y: 24 });
    await session.hold(2800);
  } finally {
    await fixture.stop();
  }
};
