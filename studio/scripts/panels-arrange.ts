// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const STAND = "Test stand";
const NEW_PANEL = "New panel";

/** pill returns the panel selector's pill for `name`. */
const pill = (session: capture.CaptureSession, name: string) =>
  session.page
    .locator(".console-panel-selector .pluto-tabs__tab")
    .filter({ hasText: name })
    .first();

/** mosaicTab returns the visible panel's tab named `name`. */
const mosaicTab = (session: capture.CaptureSession, name: string) =>
  session.page
    .locator(".console-mosaic .pluto-tabs__tab:visible")
    .filter({ hasText: name })
    .first();

/**
 * Docs `console/panels/arrange`: drag a table tab to the right edge of its panel to
 * split it beside a line plot, create a second panel from the panel selector, then
 * drag the plot tab onto that panel's name to move it there.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session);

    // A second "New panel" pill would be ambiguous, so the first gets a name.
    await pill(session, NEW_PANEL).click({ button: "right", timeout: 5000 });
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
    await page.keyboard.type(STAND);
    await page.keyboard.press("Enter");
    await session.waitFor(pill(session, STAND));

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideBottomToolbar(session);
    const tabs = page.locator(".console-mosaic .pluto-tabs__tab");
    const plotName = (await tabs.first().innerText()).trim();
    await capture.clickPanelCreate(session);
    await capture.createComponent(session, "Table");
    await session.waitFor(page.locator(".console-table").first());
    await capture.hideBottomToolbar(session);
    const tableName = (await tabs.nth(1).innerText()).trim();
    await session.settleWall(10000);
    await session.settle(500);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1800);

    // A drop in the outer third of a pane splits it.
    const leaf = await page
      .locator(".console-mosaic .pluto-mosaic__leaf")
      .first()
      .boundingBox();
    if (leaf == null) throw new Error("the panel's pane is not on screen");
    await session.drag(
      mosaicTab(session, tableName),
      { x: leaf.x + leaf.width * 0.85, y: leaf.y + leaf.height / 2 },
      { zoom: false },
    );
    await session.waitFor(page.locator(".console-mosaic .pluto-mosaic__leaf").nth(1));
    await session.hold(1800);

    await session.click(page.locator(".console-panel-selector > .pluto-btn").first(), {
      zoom: false,
    });
    await session.waitFor(pill(session, NEW_PANEL));
    await session.waitFor(page.getByText("No components open").first());
    await session.hold(1400);

    await session.click(pill(session, STAND), { text: true, zoom: false });
    await session.waitFor(mosaicTab(session, plotName));
    await session.hold(1000);

    await session.drag(mosaicTab(session, plotName), pill(session, NEW_PANEL), {
      zoom: false,
    });
    // The move follows the tab: the new panel opens with the plot in it.
    await session.waitFor(
      page
        .locator(".console-panel-selector .pluto-tabs__tab[aria-selected='true']")
        .filter({ hasText: NEW_PANEL })
        .first(),
    );
    await session.waitFor(mosaicTab(session, plotName));
    // Off the panel: the plot raises a crosshair under the pointer.
    await session.moveTo({ x: 900, y: 24 });
    await session.hold(2800);
  } finally {
    await fixture.stop();
  }
};
