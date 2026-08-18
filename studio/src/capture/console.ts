// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator, type Page } from "playwright";

import { type CaptureSession } from "@/capture/rig";

export interface Credentials {
  username: string;
  password: string;
}

/**
 * login fills the Console login form, passes the projects screen (selecting or
 * creating `project`), and waits for the workspace shell. Runs before recording
 * starts, so it fills fields directly instead of typing.
 */
export const login = async (
  session: CaptureSession,
  { username, password }: Credentials,
  project = "Docs Videos",
): Promise<void> => {
  const { page } = session;
  await session.waitFor(page.locator(".pluto-field__username input"));
  await page.locator(".pluto-field__username input").first().fill(username);
  await page.locator(".pluto-field__password input").first().fill(password);
  await page.getByRole("button", { name: "Log In" }).click();

  const palette = page.locator(".console-palette button");
  const newProject = page.getByText("New project").first();
  for (let i = 0; i < 900; i++) {
    if (await palette.isVisible().catch(() => false)) break;
    if (await newProject.isVisible().catch(() => false)) break;
    await session.settle(1000 / 60);
  }
  if (!(await palette.isVisible().catch(() => false))) {
    // The splash's project list loads asynchronously after the create button
    // renders, so give the existing card time to appear before falling back.
    const existing = page
      .locator(".console-project-splash__items [data-menu-key]")
      .filter({ hasText: project })
      .first();
    for (let i = 0; i < 300; i++) {
      if (await existing.isVisible().catch(() => false)) break;
      await session.settle(1000 / 60);
    }
    if (await existing.isVisible().catch(() => false))
      await existing.click({ timeout: 5000 });
    else {
      await newProject.click({ timeout: 5000 });
      const nameInput = page.getByPlaceholder("Name").first();
      await session.waitFor(nameInput, 300);
      await nameInput.fill(project);
      await session.settle(300);
      await page.getByRole("button", { name: "Create" }).click({ timeout: 5000 });
    }
  }
  await session.waitFor(palette, 900);
  await session.settle(1000);
};

/**
 * contextMenu right-clicks the target and selects an option from the menu that
 * opens, all as recorded input.
 */
export const contextMenu = async (
  session: CaptureSession,
  target: Parameters<CaptureSession["rightClick"]>[0],
  option: string,
): Promise<void> => {
  const { page } = session;
  await session.rightClick(target);
  const menu = page.locator(".pluto-menu-context").first();
  await session.waitFor(menu);
  await session.hold(300);
  await session.click(menu.getByText(option, { exact: true }).first());
};

/**
 * openToolbar opens a resource toolbar (e.g. "range", "task", "cluster") by
 * clicking its main-nav icon, as recorded input. No-op when already open.
 */
export const openToolbar = async (
  session: CaptureSession,
  resource: string,
): Promise<void> => {
  const { page } = session;
  const button = page
    .locator("button.console-main-nav__item")
    .filter({ has: page.locator(`svg.pluto-icon--${resource}`) })
    .first();
  if ((await button.getAttribute("class"))?.includes("selected") ?? false) return;
  await session.click(button);
  await session.waitFor(
    page.locator(".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)"),
  );
};

/**
 * resizeToolbar sets the open left drawer's width by driving its resize
 * handle with synthetic events. The width is per-rail state, so a setup-time
 * resize persists for every toolbar opened on that rail during recording.
 * Clamped by the active toolbar's min/max size.
 */
export const resizeToolbar = async (
  session: CaptureSession,
  width: number,
): Promise<void> => {
  const { page } = session;
  const drawer = page
    .locator(".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)")
    .first();
  await session.waitFor(drawer);
  await drawer.evaluate((el, target) => {
    const handle = el.querySelector(".pluto-resize__handle");
    if (handle == null) throw new Error("drawer resize handle not found");
    const rect = handle.getBoundingClientRect();
    const x0 = rect.x + rect.width / 2;
    const y = rect.y + rect.height / 2;
    const x1 = x0 + (target - el.getBoundingClientRect().width);
    handle.dispatchEvent(
      new DragEvent("dragstart", { bubbles: true, clientX: x0, clientY: y }),
    );
    window.dispatchEvent(
      new MouseEvent("mousemove", { clientX: x1, clientY: y, buttons: 1 }),
    );
    window.dispatchEvent(new MouseEvent("mouseup", { clientX: x1, clientY: y }));
  }, width);
  await session.settle(300);
};

/** closeToolbar closes the open side toolbar by clicking its selected nav icon. */
export const closeToolbar = async (session: CaptureSession): Promise<void> => {
  const { page } = session;
  const selected = page.locator("button.console-main-nav__item.pluto--selected");
  if (
    !(await selected
      .first()
      .isVisible()
      .catch(() => false))
  )
    return;
  await session.click(selected.first());
};

/**
 * clearPaletteInput empties the palette input when it retains the previous
 * query; the palette keeps its text across opens, so typing without clearing
 * appends to the stale query and searches for garbage.
 */
const clearPaletteInput = async (
  session: CaptureSession,
  input: Locator,
): Promise<void> => {
  if ((await input.inputValue().catch(() => "")) === "") return;
  await session.press("ControlOrMeta+a");
  await session.press("Delete");
};

/**
 * searchPalette opens the palette in search mode, types the query, and selects
 * the first matching result, as recorded input.
 */
export const searchPalette = async (
  session: CaptureSession,
  query: string,
  { typeSpeed = 1.5 }: CommandPaletteOptions = {},
): Promise<void> => {
  const { page } = session;
  await session.click(page.locator(".console-palette button").first());
  const input = page.locator(".console-palette__input input[role='textbox']");
  await session.waitFor(input);
  await clearPaletteInput(session, input);
  session.setSpeed(typeSpeed);
  await session.zoom(
    page.locator(".pluto-dialog__dialog:has(.console-palette__input)").first(),
  );
  await session.type(query);
  await session.hold(400);
  session.setSpeed(1);
  await session.press("Enter");
  session.endZoom();
};

/**
 * field returns the input associated with a form label, for filling or
 * clicking. Matches the Console's label-above-input form layout.
 */
export const field = (page: Page, label: string): Locator =>
  page.getByText(label, { exact: true }).locator("..").locator("input").first();

/**
 * fillField clicks the input labeled `label` and types `value` into it, as
 * recorded input.
 */
export const fillField = async (
  session: CaptureSession,
  label: string,
  value: string,
): Promise<void> => {
  await session.click(field(session.page, label));
  await session.type(value);
};

/** clickButton clicks the button with the given accessible name. */
export const clickButton = async (
  session: CaptureSession,
  label: string,
): Promise<void> => {
  await session.click(session.page.getByRole("button", { name: label }).first());
};

/** tab returns the mosaic tab with the given title. */
export const tab = (page: Page, name: string): Locator =>
  page.locator(".pluto-tabs-selector").getByText(name, { exact: true }).first();

/** selectTab clicks the mosaic tab with the given title, as recorded input. */
export const selectTab = async (
  session: CaptureSession,
  name: string,
): Promise<void> => {
  await session.click(tab(session.page, name));
};

/** closeTab closes the mosaic tab with the given title via its close button. */
export const closeTab = async (
  session: CaptureSession,
  name: string,
): Promise<void> => {
  const target = tab(session.page, name)
    .locator("..")
    .getByLabel("pluto-tabs__close")
    .first();
  await session.click(target);
  await session.waitForHidden(tab(session.page, name));
};

/**
 * treeItem returns a resource-tree item by its ontology type prefix (e.g.
 * "range:", "user:", "channel:") and display name, for clicks, drags, and
 * context menus.
 */
export const treeItem = (page: Page, prefix: string, name: string): Locator =>
  page.locator(`div[id^='${prefix}']`).filter({ hasText: name }).first();

export interface CommandPaletteOptions {
  /** Presentation speed while the query types; 1 is natural cadence. */
  typeSpeed?: number;
}

/**
 * commandPalette opens the palette and executes a command, as recorded input:
 * the palette click, query typing, and selection all land on the timeline.
 * Typing plays at `typeSpeed` and returns to natural speed before selection.
 * The camera frames the palette dialog while the query types.
 */
export const commandPalette = async (
  session: CaptureSession,
  command: string,
  { typeSpeed = 1.5 }: CommandPaletteOptions = {},
): Promise<void> => {
  const { page } = session;
  await session.click(page.locator(".console-palette button").first());
  const input = page.locator(".console-palette__input input[role='textbox']");
  await session.waitFor(input);
  await clearPaletteInput(session, input);
  session.setSpeed(typeSpeed);
  await session.type(">");
  await session.zoom(
    page.locator(".pluto-dialog__dialog:has(.console-palette__input)").first(),
  );
  await session.type(command);
  await session.hold(400);
  session.setSpeed(1);
  await session.press("Enter");
  session.endZoom();
};
