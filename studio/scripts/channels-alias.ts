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
 * Docs `console/channels/alias`: with a range active, right-click a channel in
 * the Channels toolbar, set an alias under that range, and show the toolbar
 * switch to the alias.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.seedStaticTelemetry({ channels: ["digital_input_1"], samples: 200 });
  await fixtures.seedRanges(["Hotfire 09"]);
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  // Aliases belong to the active range, so activate one before recording.
  await capture.searchPalette(session, "Hotfire 09");
  await capture.openToolbar(session, "channel");
  await capture.resizeToolbar(session, 400);
  const item = capture.treeItem(page, "channel:", "digital_input_1");
  await session.waitFor(item);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1000);

  await capture.contextMenu(session, item, "Set alias under Hotfire 09");
  // The tree name turns into a contenteditable with its text selected, so
  // typing replaces the channel name.
  const editable = page
    .locator(".pluto-text--editable[contenteditable='true']")
    .first();
  await session.waitFor(editable);
  await session.hold(400);
  await session.type("pressure_01");
  await session.hold(400);
  await session.press("Enter");

  const aliased = page
    .locator(".console-nav__drawer")
    .getByText("pressure_01", { exact: true })
    .first();
  await session.waitFor(aliased);
  await session.hold(2200);
};
