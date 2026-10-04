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
 * Docs `console/logs/create`: create a log from the component selector, click Configure
 * channels, and add a channel.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    const [channel] = fixture.channels;
    await capture.login(session);
    await capture.clearPanel(session);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1800);

    await capture.clickPanelCreate(session);
    // The selector frame fills the panel, so zoom on the center of its options.
    const options = page.locator(".console-layout-selector__frame").getByRole("button");
    await session.waitFor(options.first());
    const [first, last] = await Promise.all([
      options.first().boundingBox(),
      options.last().boundingBox(),
    ]);
    if (first == null || last == null) throw new Error("selector options are hidden");
    await session.zoom(
      {
        x: (first.x + last.x + last.width) / 2,
        y: (first.y + last.y + last.height) / 2,
      },
      1.5,
    );
    await session.hold(800);
    await capture.createComponent(session, "Log");
    session.endZoom();
    await session.waitFor(page.locator(".pluto-log").first());
    await session.hold(1000);

    await session.click(page.getByText("Configure channels", { exact: true }).first(), {
      text: true,
    });
    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    const add = drawer.getByText("Add channel", { exact: true }).first();
    await session.waitFor(add);
    await session.hold(600);
    // The toolbar spans the window, so zoom on a point over its channel list.
    const box = await add.boundingBox();
    if (box == null) throw new Error("log toolbar is not on screen");
    await session.zoom({ x: box.x + 300, y: box.y }, 1.5);
    await session.click(add, { text: true });
    const input = page.locator("input[placeholder*='Search']");
    await session.waitFor(input);
    await session.hold(400);
    await session.type(channel);
    const item = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: channel })
      .first();
    await session.waitFor(item);
    await session.hold(300);
    session.endZoom();
    await session.click(item.getByText(channel, { exact: true }).first(), {
      text: true,
    });
    // Off the log, so the new lines read without a cursor over them.
    await session.moveTo({ x: 1200, y: 300 });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
