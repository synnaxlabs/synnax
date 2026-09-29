// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, type TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

/**
 * The landing page hero still: the TPC stand's Operations panel mid-coldflow, with the
 * OX pressures regulating on an annotated plot and the Arc sequence open beside it.
 * The last captured frame is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await capture.login(session, world.project);
    // The hidden Automations panel stays mounted, so match visible nodes only.
    await session.click(page.getByText("Operations", { exact: true }).first());
    const diagram = page.locator(".pluto-diagram:visible").first();
    await session.waitFor(diagram);
    await session.settle(1500);

    // Replace the idle fuel plot with the Arc sequence that runs the stand.
    const fuel = page
      .locator(".pluto-mosaic__leaf:visible")
      .filter({ hasText: "Fuel Pressures" })
      .first();
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
    const fuelTabs = fuel.locator(".pluto-tabs__selector").first();
    const bar = await fuelTabs.boundingBox();
    if (bar == null) throw new Error("the fuel plot's tab bar is not on screen");
    await session.drag(arcTab, { x: bar.x + bar.width * 0.5, y: bar.y + bar.height / 2 });
    await session.settle(1500);

    await session.click(page.getByText("Operator", { exact: true }).first());
    await session.settle(500);
    await session.click(
      page.locator(".console-controls button:has(svg.pluto-icon--circle)").first(),
    );
    await session.settle(800);
    const start = session.now();
    await session.click(diagram.getByRole("button", { name: "Start Sequence" }).first());
    await session.waitFor(() => world.logged.includes("OX MPV open"), 3600);
    const flowing = session.now();
    await annotate(client, start, flowing);
    await session.settle(9000);

    const plot = page
      .locator(".pluto-mosaic__leaf:visible")
      .filter({ hasText: "OX Pressures" })
      .locator(".pluto-line-plot")
      .first();
    const box = await plot.boundingBox();
    if (box == null) throw new Error("the OX Pressures plot is not on screen");
    await session.moveTo({ x: box.x + box.width * 0.82, y: box.y + box.height * 0.6 });

    session.startRecording();
    await session.hold(300);
  } finally {
    await client.close();
    await world.stop();
  }
};

/** annotate marks the run's phases as ranges, which the live plot draws as bands. */
const annotate = async (
  client: ReturnType<typeof fixtures.connect>,
  start: TimeStamp,
  flowing: TimeStamp,
): Promise<void> => {
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
