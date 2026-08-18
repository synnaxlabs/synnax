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
 * login fills the Console login form and waits for the workspace shell. Runs
 * before recording starts, so it fills fields directly instead of typing.
 */
export const login = async (
  session: CaptureSession,
  { username, password }: Credentials,
): Promise<void> => {
  const { page } = session;
  await session.waitFor(page.locator(".pluto-field__username input"));
  await page.locator(".pluto-field__username input").first().fill(username);
  await page.locator(".pluto-field__password input").first().fill(password);
  await page.getByRole("button", { name: "Log In" }).click();
  await session.waitFor(page.locator(".console-palette button"));
  await session.settle(1000);
};

/**
 * commandPalette opens the palette and executes a command, as recorded input:
 * the palette click, query typing, and selection all land on the timeline.
 */
export const commandPalette = async (
  session: CaptureSession,
  command: string,
): Promise<void> => {
  const { page } = session;
  await session.click(page.locator(".console-palette button").first());
  const input = page.locator(".console-palette__input input[role='textbox']");
  await session.waitFor(input);
  await session.type(`>${command}`);
  await session.hold(400);
  await session.press("Enter");
};
