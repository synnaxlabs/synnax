// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator, type Page } from "playwright";

import { control } from "@/capture/plot";
import { type CaptureSession } from "@/capture/rig";
import { type Point } from "@/timeline";

/** Tabs of the schematic's bottom toolbar. */
export type ToolbarTab = "Symbols" | "Properties" | "Control";

/** Sides a symbol exposes attachment points on. */
export type Side = "left" | "right" | "top" | "bottom";

/** A position on the schematic canvas, in pixels from its top-left corner. */
export interface CanvasPosition {
  x: number;
  y: number;
}

/** canvas returns the schematic's drawing surface. */
export const canvas = (page: Page): Locator =>
  page.locator(".react-flow__pane").first();

/** symbols returns every symbol on the canvas, in no fixed order. */
const symbols = (page: Page): Locator => page.locator(".react-flow__node");

/**
 * handle returns one of a symbol's attachment points. A symbol can carry
 * several per side, ordered as the symbol renders them; `index` picks among
 * them.
 */
export const handle = (symbol: Locator, side: Side, index = 0): Locator =>
  symbol.locator(`.react-flow__handle-${side}`).nth(index);

/**
 * toolbarButton returns a button of the schematic's bottom toolbar by the
 * names of the icons it carries. The buttons have no accessible name of their
 * own, and the create/import pairs differ only by their corner icon.
 */
export const toolbarButton = (page: Page, ...icons: string[]): Locator => {
  let button = page.locator(".console-nav__drawer.lyra--location-bottom button");
  for (const icon of icons)
    button = button.filter({ has: page.locator(`svg.lyra-icon--${icon}`) });
  return button.first();
};

const point = async (page: Page, { x, y }: CanvasPosition): Promise<Point> => {
  const box = await canvas(page).boundingBox();
  if (box == null) throw new Error("schematic canvas is not on screen");
  return { x: box.x + x, y: box.y + y };
};

const drawer = (page: Page): Locator =>
  page.locator(".console-nav__drawer.lyra--location-bottom").first();

/**
 * selectToolbarTab opens a tab of the schematic's bottom toolbar, opening the
 * drawer first when it is closed, as recorded input. Selecting a symbol
 * switches the toolbar to Properties on its own.
 */
export const selectToolbarTab = async (
  session: CaptureSession,
  tab: ToolbarTab,
): Promise<void> => {
  const { page } = session;
  const symbolsTab = drawer(page).getByText("Symbols", { exact: true }).first();
  if (!(await symbolsTab.isVisible().catch(() => false))) {
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
    await session.waitFor(symbolsTab);
  }
  const target = drawer(page).getByText(tab, { exact: true }).first();
  await session.waitFor(target);
  await session.click(target, { text: true, zoom: false });
};

/**
 * place adds the named symbol to the canvas and drags its body onto
 * `position`, returning the new symbol. The library drops every new symbol at
 * the canvas center, so each one is moved clear before the next is added.
 * Runs as recorded input; call it before recording to stage a shot.
 */
export const place = async (
  session: CaptureSession,
  name: string,
  position: CanvasPosition,
): Promise<Locator> => {
  const { page } = session;
  await selectToolbarTab(session, "Symbols");
  const search = page.locator(".console-schematic__symbols-search input").first();
  await search.fill(name);
  const button = page
    .locator(".console-schematic-symbols__button")
    .filter({ has: page.getByText(name, { exact: true }) })
    .first();
  await session.waitFor(button);
  const ids = async (): Promise<string[]> =>
    (
      await symbols(page).evaluateAll((nodes) =>
        nodes.map((n) => n.getAttribute("data-id")),
      )
    ).filter((id): id is string => id != null);
  const before = new Set(await ids());
  await session.click(button, { zoom: false });
  // React Flow does not keep nodes in creation order, so find the new one by id.
  let id: string | undefined;
  for (let tries = 0; id == null; tries++) {
    if (tries > 200) throw new Error(`placing ${name} added no schematic node`);
    id = (await ids()).find((i) => !before.has(i));
    if (id == null) await session.settle(50);
  }
  const symbol = page.locator(`.react-flow__node[data-id="${id}"]`);
  await search.fill("");
  // Only the symbol's body starts a node drag; the node box also spans its
  // label slots.
  const body = symbol.locator(".pluto-drag-handle").first();
  await session.drag(body, await point(page, position), { zoom: false });
  return symbol;
};

/**
 * selectSymbols drags a selection box around `targets`, as recorded input. The
 * box is their combined bounds grown by `padding`, so it must have that much
 * empty canvas around it.
 */
export const selectSymbols = async (
  session: CaptureSession,
  targets: Locator[],
  padding = 48,
): Promise<void> => {
  const boxes = await Promise.all(targets.map((t) => t.boundingBox()));
  const found = boxes.filter((b) => b != null);
  if (found.length === 0) throw new Error("no symbols to select");
  const from = {
    x: Math.min(...found.map((b) => b.x)) - padding,
    y: Math.min(...found.map((b) => b.y)) - padding,
  };
  const to = {
    x: Math.max(...found.map((b) => b.x + b.width)) + padding,
    y: Math.max(...found.map((b) => b.y + b.height)) + padding,
  };
  await session.drag(from, to, { zoom: false });
};

/**
 * channelField returns the Channel field of a section, such as "State" or "Command",
 * in the selected symbol's Control tab.
 */
export const channelField = (properties: Locator, section: string): Locator =>
  properties
    .locator(".lyra-form-section")
    .filter({
      has: properties
        .page()
        .locator(".lyra-form-section__header")
        .getByText(section, { exact: true }),
    })
    .locator(".lyra-input__item")
    .filter({ hasText: "Channel" })
    .first();

/**
 * bindChannel picks `channel` for a section's Channel field in the selected symbol's
 * open properties tab. It fills the search instead of typing, so it suits setup, not
 * a recorded beat.
 */
export const bindChannel = async (
  session: CaptureSession,
  section: string,
  channel: string,
): Promise<void> => {
  const { page } = session;
  const properties = page.locator(".console-schematic__properties").first();
  const item = channelField(properties, section);
  await session.click(item.getByText("Select channel", { exact: true }).first(), {
    zoom: false,
  });
  const search = page.getByPlaceholder("Search channels...").first();
  await session.waitFor(search);
  await search.fill(channel);
  const option = page
    .locator(".lyra-list__item")
    .filter({ has: page.getByText(channel, { exact: true }) })
    .first();
  await session.waitFor(option);
  await session.click(option, { zoom: false });
  await session.settle(400);
};

/**
 * fitView fits and centers the schematic in its pane, off camera. Each click on the
 * control also toggles fit on resize, so a second click turns it back off.
 */
export const fitView = async (session: CaptureSession): Promise<void> => {
  const button = control(session.page, "expand");
  await button.click({ timeout: 5000 });
  await button.click({ timeout: 5000 });
  await session.settle(300);
};

/**
 * deselect clears the selection by clicking an empty part of the canvas, as
 * recorded input.
 */
export const deselect = async (
  session: CaptureSession,
  at: CanvasPosition,
): Promise<void> => {
  await session.click(await point(session.page, at), { zoom: false });
};
