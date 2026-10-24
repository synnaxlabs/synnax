// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan } from "@synnaxlabs/client";
import { type Locator } from "playwright";

import { capture, type fixtures } from "@/index";

/**
 * startColdflow opens the Operations panel of the TPC stand, sets it up for a still,
 * and starts a coldflow: the thermocouple table shows bare values, the Ox Coldflow Arc
 * replaces the idle fuel plot, and the operator takes control and starts the sequence.
 * Once the OX main valve opens, it marks the pressurization and the coldflow as ranges,
 * which the OX pressure plot draws as bands. The Console must be logged in to the
 * stand's project.
 */
export const startColdflow = async (
  session: capture.CaptureSession,
  world: fixtures.World,
  client: ReturnType<typeof fixtures.connect>,
): Promise<void> => {
  const { page } = session;
  const leaf = (text: string): Locator =>
    page.locator(".pluto-mosaic__leaf:visible").filter({ hasText: text }).first();
  // The hidden Automations panel stays mounted, so match visible nodes only.
  await session.click(page.getByText("Operations", { exact: true }).first());
  const diagram = page.locator(".pluto-diagram:visible").first();
  await session.waitFor(diagram);
  await session.settle(1500);

  // Indicators show while editing, so editing goes off before they hide.
  const table = leaf("Thermocouples").locator(".console-table").first();
  for (const option of ["Disable editing", "Hide indicators", "Center table"]) {
    await capture.contextMenu(session, table, option);
    session.endZoom();
    await session.settle(300);
  }

  // Replace the idle fuel plot with the Arc sequence that runs the stand.
  const fuel = leaf("Fuel Pressures");
  await session.click(fuel.getByText("Fuel Pressures", { exact: true }).first());
  await capture.openToolbar(session, "arc");
  const arc = page
    .locator(".console-nav__drawer")
    .getByText("Ox Coldflow", { exact: true })
    .first();
  await session.waitFor(arc);
  await session.doubleClick(arc);
  await session.settle(1500);
  await capture.closeToolbar(session);
  await session.settle(500);
  const arcTab = page
    .locator(".console-mosaic .pluto-tabs__tab")
    .filter({ hasText: "Ox Coldflow" })
    .first();
  const bar = await fuel.locator(".pluto-tabs__selector").first().boundingBox();
  if (bar == null) throw new Error("the fuel plot's tab bar is not on screen");
  await session.drag(arcTab, { x: bar.x + bar.width * 0.5, y: bar.y + bar.height / 2 });
  await session.settle(1500);

  await session.click(page.getByText("Operator", { exact: true }).first());
  await session.settle(500);
  await session.click(capture.control(page, "circle"));
  await session.settle(800);
  const start = session.now();
  await session.click(diagram.getByRole("button", { name: "Start Sequence" }).first());
  await session.waitFor(() => world.logged.includes("OX MPV open"), 3600);
  const flowing = session.now();
  await client.ranges.create([
    {
      name: "Pressurization",
      color: "#3774D0",
      timeRange: new TimeRange(start, flowing),
    },
    {
      name: "Coldflow",
      color: "#F5C400",
      timeRange: new TimeRange(flowing, flowing.add(TimeSpan.seconds(30))),
    },
  ]);
};
