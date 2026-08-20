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
    await capture.login(session, { username: "synnax", password: "seldon" });

    await capture.createComponent(session, "Schematic");
    await session.waitFor(page.locator(".pluto-diagram").first());
    // "Generic" is the catalog name of the plain valve; "Valve" is its label.
    const valve = await capture.place(session, "Generic", { x: 500, y: 200 });

    const properties = page.locator(".console-schematic__properties").first();
    await session.click(
      properties.locator("[role='tab']").filter({ hasText: "Control" }).first(),
      { text: true, zoom: false },
    );
    const bind = async (label: string, channel: string): Promise<void> => {
      const item = properties
        .locator(".pluto-input__item")
        .filter({ hasText: label })
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
    await bind("State channel", fixture.state);
    await bind("Command channel", fixture.command);

    await capture.deselect(session, { x: 200, y: 400 });
    // Control mode leaves the toolbar on a "not editable" placeholder, so the
    // drawer closes before recording.
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
    await session.settle(600);
    // Park the cursor inside the schematic: the floating controls only show
    // while the panel is hovered.
    await session.moveTo({ x: 300, y: 700 });

    session.startRecording();
    await session.hold(1200);

    // Wide for the acquire: the control button sits at the far corner from the
    // legend it turns on.
    await session.click(
      page.locator(".console-controls button:has(svg.pluto-icon--circle)").first(),
      { zoom: false },
    );
    await session.waitFor(page.locator(".pluto-legend-entry").first());
    await session.hold(1400);

    // An authored point zoom, not auto-zoom on the valve: the legend and the
    // valve are far apart, and only a frame this wide carries both.
    await session.moveTo({ x: 430, y: 440 });
    await session.zoom({ x: 340, y: 210 }, 1.7);
    await session.hold(900);

    await session.click(valve.locator("button.pluto-symbol-primitive-toggle").first(), {
      zoom: false,
    });
    await session.hold(400);
    // The echo runs on wall time: command out, state back through the streamer.
    await session.settleWall(600);
    await session.hold(1200);

    // Clear the cursor off the valve so the indicator stays readable.
    await session.moveTo({ x: 430, y: 440 });
    await session.hold(2600);
  } finally {
    await fixture.stop();
  }
};
