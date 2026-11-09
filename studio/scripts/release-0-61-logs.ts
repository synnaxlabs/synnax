// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const SAMPLES = 2880;
const PERIOD_MS = 60_000;

/**
 * Video for the logs section of the v0.61 release notes: a log of a channel with two days
 * of stored values opens on the newest line and scrolls back in time, past a date
 * heading. The scrollbar shows the place in time.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const channels = await fixtures.createStaticTelemetry({
    channels: ["tank_pressure"],
    samples: SAMPLES,
    periodMs: PERIOD_MS,
  });
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await capture.clickPanelCreate(session);
  await capture.createComponent(session, "Log");
  const log = page.locator(".pluto-log").first();
  await session.waitFor(log);
  const add = page.getByText("Add channel", { exact: true }).first();
  if (!(await add.isVisible().catch(() => false)))
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
  await session.waitFor(add);
  await session.click(add, { text: true, zoom: false });
  const search = page.locator("input[placeholder*='Search']");
  await session.waitFor(search);
  await search.fill(channels[0]);
  const item = page
    .locator(".lyra-list__item:not(.lyra-tree__item)")
    .filter({ hasText: channels[0] })
    .first();
  await session.waitFor(item);
  await session.click(item.getByText(channels[0], { exact: true }).first(), {
    text: true,
    zoom: false,
  });
  await capture.hideBottomToolbar(session);
  await session.settleWall(2000);
  await session.settle(800);
  const box = await log.boundingBox();
  if (box == null) throw new Error("the log is hidden");
  const center = { x: box.x + box.width / 2, y: box.y + box.height / 2 };
  await session.moveTo(center);

  session.startRecording();
  await session.hold(1800);

  for (let i = 0; i < 90; i++) {
    await page.mouse.wheel(0, -180);
    await session.hold(50);
  }
  await session.hold(2000);
  for (let i = 0; i < 60; i++) {
    await page.mouse.wheel(0, -600);
    await session.hold(40);
  }
  await session.hold(3000);
};
