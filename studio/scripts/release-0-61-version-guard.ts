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

const HEADING = "Console 0.61.0 cannot work with Core 0.60.1";

/**
 * Still for the updates section of the v0.61 release notes: the screen that a Console
 * shows when the Core runs another minor version. The page reads an old version from the
 * connectivity check of the Core. The heading names the versions in the release notes,
 * because the build under capture may carry another number.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await page.evaluate(() => {
    const { fetch } = window;
    window.fetch = async (input, init) => {
      const response = await fetch(input, init);
      const url = input instanceof Request ? input.url : String(input);
      if (!url.includes("/connectivity/check")) return response;
      const body = await response.text();
      return new Response(
        body.replace(/("node_version":\s*")[^"]+/, "$10.1.0"),
        response,
      );
    };
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
