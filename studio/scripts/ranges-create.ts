// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

// Credentials and project come from the environment; never commit them.
const USERNAME = process.env.SY_USERNAME ?? "";
const PASSWORD = process.env.SY_PASSWORD ?? "";
const PROJECT = process.env.SY_PROJECT ?? "";

/**
 * Docs `console/ranges/create`. Three ways to create a range, from the Ranges
 * toolbar, the command palette, and a plot selection.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;

  // Off-camera setup: the served console auto-connects to its own Core, so
  // setup is just login and project selection.
  await session.waitFor(page.locator(".pluto-field__username input"));
  await page.locator(".pluto-field__username input").first().fill(USERNAME);
  await page.locator(".pluto-field__password input").first().fill(PASSWORD);
  await page.getByRole("button", { name: "Log in" }).click();

  const project = page.getByText(PROJECT, { exact: true }).first();
  await session.waitFor(project, 1800);
  await project.click();
  await session.waitFor(page.locator(".console-palette button"), 900);
  await session.settle(1000);

  await capture.openToolbar(session, "range");
  await capture.resizeToolbar(session, 400);
  await session.moveTo({ x: 756, y: 500 });

  const name = page.getByPlaceholder("Name").first();

  session.startRecording();
  session.setSpeed(0.85);
  await session.hold(150);

  // Create a range from the Ranges toolbar. Linger on the create action.
  const toolbarCreate = page
    .locator(".console-nav__drawer .console-toolbar__action")
    .last();
  await session.moveTo(toolbarCreate);
  await session.hold(750);
  await capture.clickToolbarCreate(session);
  await session.waitFor(name);
  await session.hold(150);

  await session.click(name);
  await session.type("Example 1 (Toolbar)", 50);
  await session.hold(150);

  await capture.clickButton(session, "Save to Core");
  await session.waitForHidden(name);
  const created = page
    .locator(".console-nav__drawer")
    .getByText("Example 1 (Toolbar)", { exact: true })
    .first();
  await session.waitFor(created);
  await session.hold(150);

  // Create a range from the command palette. Palette input dispatches natively
  // because rig-stepped typing leaves the list unfiltered. The cursor still
  // travels on record.
  const paletteBtn = page.locator(".console-palette button").first();
  await session.moveTo(paletteBtn);
  await session.hold(750);
  await session.click(paletteBtn);
  const input = page.locator(".console-palette__input input[role='textbox']");
  await session.waitFor(input);
  await session.hold(100);
  await session.moveTo(input);
  await session.hold(120);
  for (const char of ">Create range") {
    await input.press(char === " " ? "Space" : char);
    await session.hold(10);
  }
  await session.hold(150);
  const command = page
    .locator(".pluto-list__item")
    .filter({ has: page.getByText("Create range", { exact: true }) })
    .first();
  await session.waitFor(command);
  await session.moveTo(command, { text: true });
  await command.click();

  await session.waitFor(name);
  await session.hold(150);

  await session.click(name);
  await session.type("Example 2 (Palette)", 50);
  await session.hold(150);

  await capture.clickButton(session, "Save to Core");
  await session.waitForHidden(name);
  await session.waitFor(
    page
      .locator(".console-nav__drawer")
      .getByText("Example 2 (Palette)", { exact: true })
      .first(),
  );
  await session.hold(150);

  // Create a range from a plot selection. Open the "Sine plot" through the
  // command palette, pause the rolling window, select a region, and create the
  // range from the selection menu.
  await session.click(paletteBtn);
  await session.waitFor(input);
  await session.hold(200);
  await session.moveTo(input);
  // The palette keeps the previous query across opens, so clear it before
  // searching for the plot.
  if ((await input.inputValue().catch(() => "")) !== "") {
    await session.press("ControlOrMeta+a");
    await session.press("Delete");
  }
  await session.hold(120);
  for (const char of "Sine plot") {
    await input.press(char === " " ? "Space" : char);
    await session.hold(10);
  }
  await session.hold(150);
  const plotResult = page
    .locator(".pluto-dialog__dialog .pluto-list__item")
    .filter({ hasText: "Sine plot" })
    .first();
  await session.waitFor(plotResult);
  await session.moveTo(plotResult, { text: true });
  await plotResult.click();

  // The plot shows a "fetching" screen before the canvas draws.
  await session.waitFor(page.locator(".pluto-line-plot").first(), 1800);
  await session.hold(700);

  await session.click(capture.control(page, "pause"), { zoom: false });
  await session.hold(150);
  await session.click(capture.control(page, "selection"), { zoom: false });
  await session.hold(250);

  // Horizontal drag only; roughly half the plot width; tuned on review.
  await session.drag({ x: 470, y: 395 }, { x: 980, y: 395 }, { zoom: false });
  await session.hold(150);

  // Linger on the context-menu item; it is what this flow highlights.
  await session.rightClick({ x: 725, y: 210 }, { zoom: false });
  const menu = page.locator(".pluto-menu-context").first();
  await session.waitFor(menu);
  await session.hold(300);
  const menuItem = menu
    .getByText("Create range from selection", { exact: true })
    .first();
  await session.moveTo(menuItem);
  await session.hold(750);
  await session.click(menuItem, { zoom: false });
  await session.waitFor(name);
  await session.hold(150);

  await session.click(name);
  await session.type("Example 3 (Plot)", 50);
  await session.hold(150);

  await capture.clickButton(session, "Save to Core");
  await session.waitForHidden(name);
  await session.hold(150);

  // Navigate back to the Range Explorer by its open mosaic tab.
  await session.click(capture.tab(page, "Range explorer"));
  await session.waitFor(page.locator(".console-range-explorer").first());
  await session.hold(2000);
};
