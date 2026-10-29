// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { runInNewContext } from "node:vm";

import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { beforeAll, describe, expect, it } from "vitest";

import Mark from "@/components/platform/Mark.astro";

const AGENTS: Record<string, string> = {
  macOS: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
  Windows: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
  Linux: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
};

describe("Mark", () => {
  let html: string;
  let script: string;

  beforeAll(async () => {
    const container = await AstroContainer.create();
    html = await container.renderToString(Mark);
    script = html.match(/<script[^>]*>([\s\S]*?)<\/script>/)?.[1] ?? "";
  });

  // Runs the page's script against a stand-in browser and returns the mark it leaves.
  const run = (agent: string, search: string = ""): string | undefined => {
    const root = { dataset: {} as Record<string, string> };
    runInNewContext(script, {
      document: { documentElement: root },
      navigator: { userAgent: agent },
      location: { search },
      URLSearchParams,
    });
    return root.dataset.os;
  };

  it.each(Object.entries(AGENTS))("should mark %s for its readers", (os, agent) => {
    expect(run(agent)).toBe(os);
  });

  it("should leave an unknown system unmarked", () => {
    expect(run("Mozilla/5.0 (PlayStation)")).toBeUndefined();
  });

  it("should mark the platform the URL names over the reader's own", () => {
    expect(run(AGENTS.macOS, "?platform=Windows")).toBe("Windows");
  });

  it("should keep the reader's own platform when the URL names Docker", () => {
    expect(run(AGENTS.Linux, "?platform=Docker")).toBe("Linux");
  });

  it("should mark the platform again after each client-side navigation", () => {
    expect(html).toContain("data-astro-rerun");
  });
});
