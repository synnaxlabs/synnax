// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, vi } from "vitest";

/**
 * Installs a spy in place of navigator.clipboard.writeText (an unmockable browser seam
 * with no injection point) and returns it. By default the spy resolves; pass an
 * implementation to simulate a rejection. Call from a beforeEach so each test gets a
 * fresh spy.
 */
export const stubClipboardWriteText = (
  impl: (text: string) => Promise<void> = async () => {},
): Mock => {
  const writeText = vi.fn(impl);
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText },
    configurable: true,
  });
  return writeText;
};
