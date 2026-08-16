// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type runtime } from "@synnaxlabs/x";

const USER_AGENTS: Record<runtime.OS, string> = {
  macOS: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
  Windows: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
  Linux: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
};

/**
 * pinOS pins navigator.userAgent so runtime.getOS() reports the given OS regardless
 * of the host platform.
 *
 * getOS caches its result on the first call and pluto modules call it while they
 * evaluate, so the pin must land before the spec's imports execute. A static import
 * of this file cannot guarantee that, which is why this module is not re-exported
 * from the @/testutil barrel. Call it from a vi.hoisted block via dynamic import:
 *
 *   vi.hoisted(async () => {
 *     const { pinOS } = await import("@/testutil/pinOS");
 *     pinOS("macOS");
 *   });
 */
export const pinOS = (os: runtime.OS): void => {
  Object.defineProperty(window.navigator, "userAgent", {
    value: USER_AGENTS[os],
    configurable: true,
  });
};
