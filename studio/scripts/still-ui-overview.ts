// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

import { startColdflow } from "./common/coldflow";

export const viewport = { width: 1440, height: 900 };

export const driver = true;

/** The Console toolbar behind each toolbar step of the tour, by step key. */
const TOOLBARS = {
  channel: "channel",
  range: "range",
  workspace: "project",
  device: "device",
  task: "task",
  arc: "arc",
  status: "status",
} as const;

/** The tree item each toolbar still expands, by step key. */
const EXPANDED: Partial<Record<string, string>> = {
  workspace: "TPC",
  device: "Test stand DAQ",
};

/** Alarms on the stand, which the statuses toolbar shows once they are favorites. */
const ALARMS = [
  { name: "OX tank pressure", variant: "success", message: "Within limits" },
  { name: "Fuel tank pressure", variant: "success", message: "Within limits" },
  { name: "Purge supply", variant: "warning", message: "Purge PT below 120 psi" },
  { name: "Igniter", variant: "info", message: "Disarmed" },
] as const;

/**
 * Stills for the Console tour on docs page `reference/get-started/ui-overview`: the TPC
 * stand's Operations panel mid-coldflow, then the same panel with the palette, each
 * toolbar, and the component toolbar open. Each still records the box of every tour
 * step it shows, keyed by the step.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await client.statuses.set(
      ALARMS.map(({ name, variant, message }) => ({
        key: `alarm:${name.toLowerCase().replaceAll(" ", "-")}`,
        name,
        variant,
        message,
      })),
    );
    await capture.login(session, world.project);

    await favorite(session, "Open range explorer", [
      "Coldflow 9",
      "Coldflow 10",
      "Coldflow 11",
    ]);
    await favorite(
      session,
      "Open status explorer",
      ALARMS.map((a) => a.name),
    );

    await startColdflow(session, world, client);
    const leaf = (text: string): Locator =>
      page.locator(".pluto-mosaic__leaf:visible").filter({ hasText: text }).first();
    await session.waitFor(() => world.logged.includes("OX Vent open"), 3600);
    // Held as the tank vents, the plot shows the whole coldflow in every still.
    await session.settle(2500);
    await session.click(capture.control(leaf("OX Pressures"), "pause"), {
      zoom: false,
    });

    const tab = (name: string): Locator =>
      page
        .locator(".pluto-mosaic__leaf:visible")
        .filter({
          has: page.locator(".lyra-tabs__tab").getByText(name, { exact: true }),
        })
        .first();
    const navItem = (toolbar: string): Locator =>
      page
        .locator("button.console-main-nav__item")
        .filter({ has: page.locator(`svg.lyra-icon--${toolbar}`) })
        .first();
    const paletteButton = page.locator(".console-palette button").first();
    const componentItem = page.locator("button.console-main-nav__item").last();
    const help = page.locator(".console-docs__open-button").first();
    // Each still records every part it shows, so the tour can jump to any of them. The
    // palette covers the parts, so its still records only the palette and its button.
    const parts = (): Record<string, Locator | Locator[]> => ({
      mosaic: page.locator(".console-mosaic").first(),
      schematic: tab("Operator"),
      lineplot: tab("OX Pressures"),
      editor: tab("Ox Coldflow"),
      project: page.locator(".console-project-selector__trigger").first(),
      search: paletteButton,
      toolbars: page.locator(
        ".lyra-navbar__content:has(.console-palette) button.console-main-nav__item",
      ),
      ...Object.fromEntries(
        Object.entries(TOOLBARS).map(([step, toolbar]) => [step, navItem(toolbar)]),
      ),
      component: componentItem,
      panels: page.locator(".console-panel-selector").first(),
      help,
    });
    // In the empty top bar left of help, so nothing on screen shows its hover state.
    const rest = async (): Promise<void> => {
      const box = await help.boundingBox();
      if (box == null) throw new Error("the help button is not on screen");
      await session.moveTo({ x: box.x - 30, y: box.y + box.height / 2 });
      await session.settle(1200);
    };

    await rest();
    await session.still("mosaic", parts());

    const palette = await capture.openPalette(session);
    await session.type("ox");
    await rest();
    await session.still("search", { search: paletteButton, palette });
    await page.keyboard.press("Escape");
    await session.settle(500);

    const left = page.locator(".console-nav__drawer.lyra--location-left").first();
    for (const [step, toolbar] of Object.entries(TOOLBARS)) {
      await capture.openToolbar(session, toolbar);
      await session.waitFor(left);
      const expand = EXPANDED[step];
      if (expand != null)
        await session.click(left.getByText(expand, { exact: true }).first(), {
          text: true,
          zoom: false,
        });
      await rest();
      await session.still(step, { ...parts(), drawer: left });
      await capture.closeToolbar(session);
      await session.settle(500);
    }

    await session.click(
      tab("OX Pressures").getByText("OX Pressures", { exact: true }).first(),
    );
    await capture.showVisualizationToolbar(session);
    const bottom = page.locator(".console-nav__drawer.lyra--location-bottom").first();
    await session.waitFor(bottom);
    await rest();
    await session.still("component", { ...parts(), drawer: bottom });
  } finally {
    await client.close();
    await world.stop();
  }
};

/**
 * favorite opens an explorer from the palette, favorites each named item in it, and
 * closes the explorer's tab.
 */
const favorite = async (
  session: capture.CaptureSession,
  command: string,
  names: readonly string[],
): Promise<void> => {
  const { page } = session;
  await capture.commandPalette(session, command);
  for (const name of names) {
    const item = page.locator(".lyra-list__item").filter({ hasText: name }).first();
    await session.waitFor(item);
    await capture.contextMenu(
      session,
      item.getByText(name, { exact: true }).first(),
      "Favorite",
    );
    session.endZoom();
    await session.settle(300);
  }
  const tab = page.locator(".console-mosaic .lyra-tabs__tab.lyra--selected").first();
  await session.click(tab.locator(".lyra-tabs__close").first(), { zoom: false });
  await session.settle(500);
};
