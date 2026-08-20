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
 * Docs `console/ranges/create-child`: on a range's overview page, click the
 * "+" in the Child ranges section, fill the dialog (parent arrives pre-set),
 * and save; the child appears in the list.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.seedRanges(["Hotfire 09"]);
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  await capture.searchPalette(session, "Hotfire 09");
  const section = page.locator(".pluto-header").filter({ hasText: "Child ranges" });
  await session.waitFor(section.first());
  await session.moveTo({ x: 756, y: 400 });

  session.startRecording();
  await session.hold(1000);

  await session.click(section.locator("button").first());
  // The overview's own title input also has the "Name" placeholder; scope to
  // the modal or the first match dismisses it.
  const name = page.locator(".console-modal").getByPlaceholder("Name").first();
  await session.waitFor(name);
  await session.hold(600);
  await session.zoom(page.locator(".console-modal").first());

  await session.click(name);
  await session.type("Ignition Sequence");
  await session.hold(600);

  await capture.clickButton(session, "Save to Core");
  await session.waitForHidden(name);
  session.endZoom();
  await session.waitFor(page.getByText("Ignition Sequence", { exact: true }).first());
  await session.hold(2200);
};
