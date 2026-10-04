// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

/**
 * Docs `console/schematics/valve`: acquire control on a schematic, click the
 * valve to command it open, and watch the control indicator, the legend, and
 * the valve's own state follow the echoed state channel.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.echoValve();
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Schematic");
    await session.waitFor(page.locator(".pluto-diagram").first());
    // "Generic" is the catalog name of the plain valve; "Valve" is its label.
    const valve = await capture.place(session, "Generic", { x: 500, y: 200 });

    const properties = page.locator(".console-schematic__properties").first();
    await session.click(
      properties.locator("[role='tab']").filter({ hasText: "Control" }).first(),
      { text: true, zoom: false },
    );
    const bind = async (section: string, channel: string): Promise<void> => {
      const item = properties
        .locator(".pluto-form-section")
        .filter({
          has: page
            .locator(".pluto-form-section__header")
            .getByText(section, { exact: true }),
        })
        .locator(".pluto-input__item")
        .filter({ hasText: "Channel" })
        .first();
      await session.click(item.getByText("Select channel", { exact: true }).first(), {
        zoom: false,
      });
      const search = page.getByPlaceholder("Search channels...").first();
      await session.waitFor(search);
      await search.fill(channel);
      const option = page
        .locator(".pluto-list__item")
        .filter({ has: page.getByText(channel, { exact: true }) })
        .first();
      await session.waitFor(option);
      await session.click(option, { zoom: false });
      await session.settle(400);
    };
    await bind("State", fixture.state);
    await bind("Command", fixture.command);

    await capture.deselect(session, { x: 100, y: 290 });
    // Control mode leaves the toolbar on a "not editable" placeholder, so the
    // drawer closes before recording.
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
    await session.settle(600);
    // The canvas only reaches its recorded size once the drawer closes.
    const pane = await capture.canvas(page).boundingBox();
    if (pane == null) throw new Error("schematic canvas is not on screen");
    await session.drag(
      valve.locator(".pluto-drag-handle").first(),
      { x: pane.x + pane.width / 2, y: pane.y + pane.height / 2 },
      { zoom: false },
    );
    await session.settle(400);
    // The drag selects the valve.
    await capture.deselect(session, { x: 100, y: 290 });
    await session.settle(300);
    // Park the cursor inside the schematic: the floating controls only show
    // while the panel is hovered.
    await session.moveTo({ x: 300, y: 560 });

    session.startRecording();
    await session.hold(1800);

    await session.click(
      page.locator(".console-controls button:has(svg.pluto-icon--circle)").first(),
      { zoom: false },
    );
    await session.waitFor(page.locator(".pluto-legend-entry").first());
    await session.hold(1400);

    // The valve's rect alone would zoom past the render's resolution.
    await session.zoom(valve, 1.6);
    await session.hold(900);
    await session.click(valve.locator("button.pluto-symbol-primitive-toggle").first());
    session.endZoom();
    await session.hold(400);
    // The echo runs on wall time: command out, state back through the streamer.
    await session.settleWall(600);
    const body = await valve.boundingBox();
    if (body == null) throw new Error("valve is hidden");
    // Clear the cursor off the valve so the indicator stays readable.
    await session.moveTo({
      x: body.x + body.width + 160,
      y: body.y + body.height + 90,
    });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
