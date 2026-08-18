// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
