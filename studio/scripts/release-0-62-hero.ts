// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  panel,
  type ranger,
  type Synnax,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { dropTemporal, pinClock, runAhead } from "./common/clock";
import { fitOperator, openOperator, operationsLayout } from "./common/coldflow";

export const viewport = { width: 1280, height: 800 };

// A range lists its labels in key order, so fixed keys hold the order on both themes.
const LABELS = {
  LOX: { key: "00000001-0000-4000-8000-000000000000", color: "#3B7DDB" },
  Nominal: { key: "00000002-0000-4000-8000-000000000000", color: "#10B061" },
  Anomaly: { key: "00000003-0000-4000-8000-000000000000", color: "#DC1360" },
};

type Label = keyof typeof LABELS;

const CAMPAIGN = "LOX campaign";
const TEST = "Coldflow 12";

/** The stand's earlier tests, with their labels. */
const TESTS: ReadonlyArray<readonly [name: string, labels: Label[]]> = [
  ["Coldflow 9", ["LOX", "Nominal"]],
  ["Coldflow 10", ["LOX", "Anomaly"]],
  ["Coldflow 11", ["LOX", "Nominal"]],
];

/** Ranges starred before the test, so the toolbar lists them under Favorites. */
const FAVORITES = ["Coldflow 10", "Coldflow 11"];

/** Each TPC panel's icon, and its color when it does not use the theme's. */
const ICONS: Record<string, { icon: string; color?: string }> = {
  Automations: { icon: "arc", color: "#A855F7" },
  Fuel: { icon: "flame", color: "#FF9B42" },
  Hardware: { icon: "rack" },
  Operations: { icon: "rocket", color: "#3774D0" },
  Oxidizer: { icon: "snowflake", color: "#0FA3A3" },
  Pressurant: { icon: "gas-cylinder", color: "#F4CA25" },
  Review: { icon: "line-plot", color: "#50C878" },
};

/** Tabs taken out of the Operations panel, so the pressure plot spans its width. */
const CLOSED = [
  ["Fuel Pressures", "lineplot"],
  ["Thermocouples", "table"],
] as const;

/**
 * Each tab moved, the tab whose leaf it splits, and the edge of that leaf it takes. The
 * plot ends on top, over the schematic and the log.
 */
const MOVES = [
  ["Operator", "OX Pressures", "bottom"],
  ["Sequence Logs", "Operator", "right"],
] as const;

/** closeTab removes the tab of resource `name` from the Operations panel. */
const closeTab = async (
  client: Synnax,
  name: string,
  type: (typeof CLOSED)[number][1],
): Promise<void> => {
  const operations = (await client.panels.retrieve({ searchTerm: "Operations" })).find(
    (p) => p.name === "Operations",
  );
  const found = (
    await client.resources.search({ searchTerm: name, types: [type] })
  ).find((r) => r.name === name);
  if (operations == null || found == null)
    throw new Error(`the TPC world has no Operations panel or "${name}"`);
  const tab = panel.findTabByResource(operations.root, found.id);
  if (tab == null) throw new Error(`the Operations panel has no "${name}" tab`);
  if (
    !(await client.panels.dispatch(operations.key, panel.removeTab({ key: tab.key })))
  )
    throw new Error("the Operations panel is not cached");
};

/**
 * Still for the hero of the v0.62 release notes: the TPC stand mid-coldflow. The ranges
 * toolbar lists the running campaign, its running test, and the test's running phase,
 * over starred and recent tests. The OX pressure plot draws the test and its phases in
 * the strip above the live pressures.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await runAhead(session);
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await client.labels.create(
      Object.entries(LABELS).map(([name, l]) => ({ name, ...l })),
    );
    const label = async (range: ranger.Range, labels: Label[]): Promise<void> =>
      await client.labels.label(
        range.resourceID,
        labels.map((l) => LABELS[l].key),
      );
    const tests = await Promise.all(
      TESTS.map(async ([name, labels]) => {
        const range = await client.ranges.retrieve(name);
        await label(range, labels);
        return range;
      }),
    );
    const campaign = await client.ranges.create({
      name: CAMPAIGN,
      color: "#7C5CFF",
      timeRange: new TimeRange(
        tests[0].timeRange.start.sub(TimeSpan.minutes(12)),
        TimeStamp.MAX,
      ),
    });
    await label(campaign, ["LOX"]);

    for (const { key, name } of await client.panels.retrieve({})) {
      const look = ICONS[name];
      if (look == null) continue;
      const actions = [panel.setIcon({ icon: look.icon })];
      if (look.color != null)
        actions.push(panel.setColor({ color: { type: "fixed", value: look.color } }));
      if (!(await client.panels.dispatch(key, actions)))
        throw new Error(`the ${name} panel is not cached`);
    }
    for (const [name, type] of CLOSED) await closeTab(client, name, type);
    const layout = await operationsLayout(client);
    for (const [name, beside, location] of MOVES)
      await layout.move(name, beside, location);
    await layout.resize(panel.ROOT_NODE_KEY, 0.4);
    await layout.resize((await layout.path("Operator")) >> 1, 0.66);

    await pinClock(session);
    await capture.login(session, world.project);
    await dropTemporal(page);
    const diagram = page.locator(".pluto-diagram:visible").first();
    await openOperator(session);
    await capture.control(diagram, "edit-off").click();
    await session.settle(1500);

    await capture.openToolbar(session, "range");
    await capture.resizeToolbar(session, 330);
    const toolbar = page.locator(".console-nav__drawer.lyra--location-left").first();
    for (const name of FAVORITES) {
      const item = toolbar.getByText(name, { exact: true }).first();
      await session.waitFor(item);
      await capture.contextMenu(session, item, "Favorite");
      session.endZoom();
      await session.settle(300);
    }
    await fitOperator(session);

    await session.click(capture.control(diagram, "circle"));
    await session.settle(800);
    const start = session.now();
    const test = await client.ranges.create({
      name: TEST,
      color: "#3B7DDB",
      timeRange: new TimeRange(start, TimeStamp.MAX),
      parent: campaign.key,
    });
    await label(test, ["LOX"]);
    await session.click(
      diagram.getByRole("button", { name: "Start Sequence" }).first(),
    );
    await session.waitFor(() => world.logged.includes("OX MPV open"), 3600);
    const flowing = session.now();
    await client.ranges.create([
      {
        name: "Pressurization",
        color: "#3774D0",
        timeRange: new TimeRange(start, flowing),
        parent: test.key,
      },
      {
        name: "Flow",
        color: "#F5C400",
        timeRange: new TimeRange(flowing, TimeStamp.MAX),
        parent: test.key,
      },
    ]);
    // Released, the schematic's controls fade out instead of staying pinned.
    await capture.control(diagram, "circle").click();
    await fitOperator(session);
    // The tank pressure cycles in its TPC band a few seconds after the valve opens.
    await session.settle(5000);
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};
