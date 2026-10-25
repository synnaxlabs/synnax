// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

const RANGE = "Hotfire 09";
const CHANNELS = ["tank_pressure", "tank_temperature"];

/**
 * Docs `console/ranges/export`: on a range overview, click the CSV icon and select
 * channels in the download dialog. The shot ends with Download enabled and saves no
 * file.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.createStaticTelemetry({ channels: CHANNELS });
  // Covers the static block, which ends 5 seconds before now.
  const now = TimeStamp.now();
  await fixtures.createRanges([
    {
      name: RANGE,
      timeRange: new TimeRange(
        now.sub(TimeSpan.seconds(130)),
        now.sub(TimeSpan.seconds(3)),
      ),
    },
  ]);
  const { page } = session;
  await capture.login(session);

  await capture.searchPalette(session, RANGE);
  const csv = page
    .locator(".console-range-overview__actions button:has(.pluto-icon--csv)")
    .first();
  await session.waitFor(csv);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await session.click(csv);
  const modal = page.locator(".console-download-csv").first();
  await session.waitFor(modal);
  await session.hold(600);
  await session.zoom(modal);
  await session.hold(400);

  await session.click(modal.getByText("Select channels to download", { exact: true }), {
    text: true,
  });
  const search = page.locator(".pluto-select__dialog:visible input").first();
  await session.waitFor(search);
  await session.hold(400);
  // The multi-select stays open across picks and keeps its search text.
  for (const [i, channel] of CHANNELS.entries()) {
    if (i > 0) {
      // Picking an item moves focus off the search input.
      await session.click(search, { zoom: false });
      await session.press("ControlOrMeta+a");
    }
    await session.type(channel);
    const item = page
      .locator(".pluto-select__dialog:visible .pluto-list__item")
      .filter({ hasText: channel })
      .first();
    await session.waitFor(item);
    await session.hold(300);
    await session.click(item.getByText(channel, { exact: true }).first(), {
      text: true,
    });
    await session.hold(300);
  }
  await session.press("Escape");
  await session.hold(500);

  // Rest on the Download button, now enabled.
  await session.moveTo(modal.getByRole("button", { name: /^Download/ }).first());
  await session.hold(1000);
  session.endZoom();
  await session.hold(3000);
};
