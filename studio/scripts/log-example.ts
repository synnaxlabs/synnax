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
 * Docs `console/logs/example`: create a log, add streaming channels in the
 * Channels tab, and give the timestamps sub-second precision; lines arrive as
 * telemetry does.
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
    await capture.createComponent(session, "Log");
    await session.waitFor(page.locator(".pluto-log").first());
    await session.hold(1200);

    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    const add = page.getByText("Add channel", { exact: true }).first();
    if (!(await add.isVisible().catch(() => false)))
      await session.click(page.locator(".console-main-nav__item").last(), {
        zoom: false,
      });
    await session.waitFor(add);
    await session.hold(600);

    for (const channel of fixture.channels) {
      await session.click(page.getByText("Add channel", { exact: true }).first(), {
        text: true,
        zoom: false,
      });
      const input = page.locator("input[placeholder*='Search']");
      await session.waitFor(input);
      await session.hold(400);
      await session.type(channel);
      const item = page
        .locator(".pluto-list__item:not(.pluto-tree__item)")
        .filter({ hasText: channel })
        .first();
      await session.waitFor(item);
      await session.click(item.getByText(channel, { exact: true }).first(), {
        text: true,
      });
      await session.hold(800);
    }

    await session.click(drawer.getByText("Properties", { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    const precision = capture.field(page, "Receipt timestamp precision");
    await session.waitFor(precision);
    await session.hold(600);
    await session.click(precision, { zoom: false });
    await session.press("ControlOrMeta+a");
    await session.type("3");
    await session.press("Enter");
    await session.hold(4000);
  } finally {
    await fixture.stop();
  }
};
