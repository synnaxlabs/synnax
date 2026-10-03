// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { QUERY } from "@/components/platform/choice";

const AGENTS: Record<string, string> = {
  macOS: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
  Windows: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
  Linux: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
};

describe("platform choice", () => {
  afterEach(() => vi.restoreAllMocks());

  it.each(Object.entries(AGENTS))(
    "should show %s first to its readers",
    (os, agent) => {
      vi.spyOn(navigator, "userAgent", "get").mockReturnValue(agent);
      expect(QUERY.initial?.()).toBe(os);
    },
  );

  it("should leave the first tab for an unknown system", () => {
    vi.spyOn(navigator, "userAgent", "get").mockReturnValue(
      "Mozilla/5.0 (PlayStation)",
    );
    expect(QUERY.initial?.()).toBeNull();
  });
});
