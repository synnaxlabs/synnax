// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

/** The projects screen, which a Desktop build opens on in place of the login form. */
const newProject = (session: capture.CaptureSession) =>
  session.page.getByText("New project").first();

/**
 * openProject opens the named project from the projects screen of Synnax Desktop, or
 * creates it when it does not exist. Runs before recording.
 */
export const openProject = async (
  session: capture.CaptureSession,
  project: string = "My project",
): Promise<void> => {
  const { page } = session;
  await session.waitFor(newProject(session), 1800);
  const existing = page
    .locator(".console-project-splash__items [data-menu-key]")
    .filter({ hasText: project })
    .first();
  // The list of projects loads after the create button renders.
  for (let i = 0; i < 300; i++) {
    if (await existing.isVisible().catch(() => false)) break;
    await session.settle(1000 / 60);
  }
  if (await existing.isVisible().catch(() => false))
    await existing.click({ timeout: 5000 });
  else {
    await newProject(session).click({ timeout: 5000 });
    const name = page.locator(capture.MODAL).getByPlaceholder("Name").first();
    await session.waitFor(name);
    await name.fill(project);
    await page.getByRole("button", { name: "Create", exact: true }).click();
  }
  await session.waitFor(page.locator(".console-palette button"), 1800);
  await session.settle(1000);
};
