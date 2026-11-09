// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

export const viewport = { width: 1280, height: 800 };

/** The older minor version that the check of the Core reports. */
const lowerMinor = (version: string): string => {
  const [major, minor, ...rest] = version.split(".");
  return [major, String(Number(minor) - 1).padStart(minor.length, "0"), ...rest].join(
    ".",
  );
};

const HEADING = "Console 0.61.0 cannot work with Core 0.60.1";

/**
 * Still for the updates section of the v0.61 release notes: the screen that a Console
 * shows when the Core runs another minor version. The Core answers its connectivity
 * check with an older version. The heading names the versions in the release notes,
 * because the build under capture may carry another number.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await page.route("**/connectivity/check*", async (route) => {
    const response = await route.fetch();
    const body = (await response.body())
      .toString("latin1")
      .replace(
        /(node_?[vV]ersion[^0-9]{0,3})(\d+\.\d+\.\d+)/,
        (_, key: string, version: string) => `${key}${lowerMinor(version)}`,
      );
    await route.fulfill({ response, body: Buffer.from(body, "latin1") });
  });
  const username = page.locator(".lyra-field__username input").first();
  await session.waitFor(username);
  await username.fill(capture.CREDENTIALS.username);
  await page
    .locator(".lyra-field__password input")
    .first()
    .fill(capture.CREDENTIALS.password);
  await page.getByRole("button", { name: "Log In" }).click();
  const heading = page.getByText("cannot work with Core").first();
  await session.waitFor(heading, 1800);
  await heading.evaluate((el, text) => {
    el.textContent = text;
  }, HEADING);
  await session.moveTo({ x: 1240, y: 760 });
  await session.settle(1000);
  await session.still("window");
};
