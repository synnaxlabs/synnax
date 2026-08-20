// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

const PROJECT = capture.PROJECT;
const CORE_NAME = "Local Core";

/** Mirrors DEV_DETACH_KEY in console/src/platform/cluster/detectConnection.ts. */
const DETACH_KEY = "synnax-dev-connection-detach";

/**
 * createProject gives the Core a project to open, so the shot ends in the
 * Console instead of on the project picker.
 */
const createProject = async (port: number): Promise<void> => {
  const client = new Synnax({
    host: "localhost",
    port,
    username: "synnax",
    password: "seldon",
  });
  try {
    await client.projects.create({ name: PROJECT });
  } finally {
    client.close();
  }
};

const coreItem = (session: capture.CaptureSession, name: string) =>
  session.page.locator(".console-cluster-list-item").filter({ hasText: name }).first();

/**
 * removeCore drops a Core from the list off camera. The session ships with a
 * "Local" and a "Demo" entry, neither of which answers on the capture's port.
 */
const removeCore = async (
  session: capture.CaptureSession,
  name: string,
): Promise<void> => {
  const item = coreItem(session, name);
  await item.click({ button: "right", timeout: 5000 });
  const menu = session.page.locator(".pluto-menu-context").first();
  await session.waitFor(menu);
  await menu
    .getByText(/^Remove/)
    .first()
    .click({ timeout: 5000 });
  await session.waitForHidden(item);
};

/**
 * Docs `console/clusters/connect`: add a Core from the Cores list, fill its
 * name, host, and port, connect, then log in and land in the Console with the
 * connection indicator reporting the Core reachable.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const port = fixtures.defaultPort();
  await createProject(port);

  // The browser build connects to the Core that serves it, so it boots straight
  // to credentials. Detaching gives it the Cores list the desktop build shows.
  await page.evaluate(`localStorage.setItem(${JSON.stringify(DETACH_KEY)}, "1")`);
  await page.reload();

  const addCore = page
    .locator(".console-create-list-item")
    .filter({ hasText: "Add a Core" })
    .first();
  await session.waitFor(addCore);
  await removeCore(session, "Demo");
  await removeCore(session, "Local");
  await session.settle(1500);
  await session.moveTo({ x: 756, y: 760 });

  session.startRecording();
  await session.hold(1200);

  // No zoom: the dialog opens over the button.
  await session.click(addCore, { zoom: false });
  const modal = page.locator(".console-modal").first();
  await session.waitFor(modal);
  await session.hold(600);
  await session.zoom(modal);

  await session.click(modal.getByPlaceholder("Synnax Core").first());
  await session.type(CORE_NAME);
  await session.hold(300);
  await session.click(modal.getByPlaceholder("localhost").first());
  await session.type("localhost");
  await session.hold(300);
  await session.click(modal.getByPlaceholder("9090").first());
  await session.type(String(port));
  await session.hold(500);

  await capture.clickButton(session, "Connect");
  await session.waitForHidden(modal);
  session.endZoom();

  const item = coreItem(session, CORE_NAME);
  await session.waitFor(item);
  await session.hold(2000);
  await session.click(item.locator(".console-cluster-list-item__name").first(), {
    zoom: false,
    text: true,
  });

  const username = page.locator(".pluto-field__username input").first();
  await session.waitFor(username);
  await session.hold(700);
  await session.click(username);
  await session.type("synnax");
  await session.hold(300);
  await session.click(page.locator(".pluto-field__password input").first());
  await session.type("seldon");
  await session.hold(500);
  await session.click(page.getByRole("button", { name: "Log in" }).first(), {
    zoom: false,
  });

  const project = page
    .locator(".console-project-splash__items [data-menu-key]")
    .filter({ hasText: PROJECT })
    .first();
  await session.waitFor(project);
  await session.hold(1200);
  await session.click(project, { zoom: false, text: true });

  await session.waitFor(page.locator(".console-palette button").first());
  await session.hold(1600);
  // Under the indicator, not on it: the cursor sprite draws down and right of
  // its point, so aiming at the badge would cover it.
  await session.moveTo({ x: 1420, y: 110 });
  await session.hold(2200);
};
