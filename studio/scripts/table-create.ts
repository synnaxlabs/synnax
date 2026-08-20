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
 * Docs `console/tables/create`: from an empty panel, open the component
 * selector with "+" and pick Table, then select a cell, switch it to a value,
 * and bind a channel; the cell starts showing live telemetry.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    const project = `Docs Videos ${Date.now().toString(36)}`;
    await capture.login(session, { username: "synnax", password: "seldon" }, project);
    await capture.clearPanel(session);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1200);

    await capture.clickPanelCreate(session);
    await session.hold(800);
    await capture.createComponent(session, "Table");
    const cell = page.locator(".pluto-table__cell").first();
    await session.waitFor(cell);
    await session.hold(1200);

    await session.click(cell, { zoom: false });
    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    const label = page.getByText("Variant", { exact: true }).first();
    if (!(await label.isVisible().catch(() => false)))
      await session.click(page.locator(".console-main-nav__item").last(), {
        zoom: false,
      });
    await session.waitFor(label);
    await session.hold(600);

    const variant = drawer.getByText("Text", { exact: true }).first();
    await session.click(variant, { text: true, zoom: false });
    const value = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: "Value" })
      .first();
    await session.waitFor(value);
    await session.click(value.getByText("Value", { exact: true }).first(), {
      text: true,
    });

    await session.click(drawer.getByText("Telemetry", { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    await session.click(page.getByText("Select channel", { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    const input = page.locator("input[placeholder*='Search']");
    await session.waitFor(input);
    await session.hold(600);
    await session.type("demo");
    const channel = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: fixture.channels[0] })
      .first();
    await session.waitFor(channel);
    await session.click(
      channel.getByText(fixture.channels[0], { exact: true }).first(),
      { text: true },
    );
    await session.press("Escape");
    await session.hold(4000);
  } finally {
    await fixture.stop();
  }
};
