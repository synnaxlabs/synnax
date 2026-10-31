// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, panel, TimeRange, TimeSpan } from "@synnaxlabs/client";
import { type Locator } from "playwright";

import { capture, type fixtures } from "@/index";

const PANEL = "Operations";

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
  await session.click(page.getByText(PANEL, { exact: true }).first());
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
    .locator(".console-mosaic .lyra-tabs__tab")
    .filter({ hasText: "Ox Coldflow" })
    .first();
  const bar = await fuel.locator(".lyra-tabs__selector").first().boundingBox();
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

/** Layout rearranges the tabs of the TPC stand's Operations panel through the Core. */
export interface Layout {
  /** move puts tab `name` on the `location` edge of the leaf that holds `beside`. */
  move: (
    name: string,
    beside: string,
    location: panel.MoveTabPayload["location"],
  ) => Promise<void>;
  /** path returns the path of the leaf that holds tab `name`. */
  path: (name: string) => Promise<number>;
  /** resize sets the size of the split at path `split`, from 0 to 1. */
  resize: (split: number, size: number) => Promise<void>;
}

/** operationsLayout returns a {@link Layout} of the TPC stand's Operations panel. */
export const operationsLayout = async (
  client: ReturnType<typeof fixtures.connect>,
): Promise<Layout> => {
  const resources = await client.ontology.retrieve({
    types: ["panel", "lineplot", "schematic", "table", "log"],
  });
  const id = (name: string): ontology.ID => {
    const resource = resources.find((r) => r.name === name);
    if (resource == null) throw new Error(`the TPC world has no "${name}"`);
    return resource.id;
  };
  const key = id(PANEL).key;
  const leaf = async (name: string): Promise<{ tab: string; path: number }> => {
    const { root } = await client.panels.retrieve(key);
    const tab = panel.findTabByResource(root, id(name));
    const path = tab == null ? undefined : panel.tabLeafPath(root, tab.key);
    if (tab == null || path == null) throw new Error(`${PANEL} has no "${name}" tab`);
    return { tab: tab.key, path };
  };
  const dispatch = async (action: panel.Action): Promise<void> => {
    if (!(await client.panels.dispatch(key, action)))
      throw new Error(`the ${PANEL} panel is not cached`);
  };
  return {
    move: async (name, beside, location) => {
      const { tab } = await leaf(name);
      const { path } = await leaf(beside);
      await dispatch(panel.moveTab({ key: tab, targetLeaf: path, location }));
    },
    path: async (name) => (await leaf(name)).path,
    resize: async (split, size) => await dispatch(panel.resizeSplit({ split, size })),
  };
};

/**
 * openOperator opens the Operator schematic of the Operations panel, with its control
 * legend and the bottom toolbar hidden.
 */
export const openOperator = async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const drawer = page
    .locator(".console-nav__drawer.lyra--visible.lyra--location-bottom")
    .first();
  // The legend switch shows only while editing, and taking control ends editing.
  await session.click(page.getByText(PANEL, { exact: true }).first());
  await session.waitFor(page.locator(".pluto-diagram:visible").first());
  await session.click(page.getByText("Operator", { exact: true }).first());
  await capture.selectToolbarTab(session, "Control");
  const legend = drawer.locator(".console-schematic__control input[type=checkbox]");
  await session.waitFor(legend.locator(".."));
  if (await legend.isChecked()) await legend.click({ force: true });
  await capture.hideBottomToolbar(session);
};

/** fitOperator fits the Operator schematic to its leaf once. */
export const fitOperator = async (session: capture.CaptureSession): Promise<void> => {
  const diagram = session.page.locator(".pluto-diagram:visible").first();
  const expand = capture.control(diagram, "expand");
  // Each click also toggles fit on resize, so a second click turns it back off.
  await expand.click();
  await expand.click();
  await session.settle(300);
};

/**
 * zoomPressures pauses the OX Pressures plot and zooms its time axis to the span from
 * `from`, a fraction of the axis, to now.
 */
export const zoomPressures = async (
  session: capture.CaptureSession,
  from: number,
): Promise<void> => {
  const plot = session.page
    .locator(".pluto-mosaic__leaf:visible")
    .filter({ hasText: "OX Pressures" })
    .first();
  await session.click(capture.control(plot, "pause"), { zoom: false });
  // A drag flatter than the zoom threshold zooms only the time axis. The axis runs from
  // the y axis to the plot's right padding.
  const box = await plot.boundingBox();
  if (box == null) throw new Error("the OX Pressures plot is not on screen");
  const start = box.x + 57;
  const end = box.x + box.width - 14;
  const y = box.y + box.height / 2;
  await session.drag(
    { x: start + (end - start) * from, y },
    { x: end, y },
    { zoom: false },
  );
};
