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
  let script: string;

  beforeAll(async () => {
    const container = await AstroContainer.create();
    const html = await container.renderToString(Mark);
    script = html.slice(html.indexOf(">") + 1, html.lastIndexOf("</script>"));
  });

  interface Page {
    /** The mark the script leaves on the page it loads in. */
    os: string | undefined;
    /** Fires the router's swap to the URL and returns the mark on the next page. */
    swapTo: (url: string) => string | undefined;
  }

  // Runs the page's script against a stand-in browser.
  const load = (agent: string, href: string = "https://docs.synnaxlabs.com/"): Page => {
    const root = { dataset: {} as Record<string, string> };
    const listeners = new Map<string, (e: unknown) => void>();
    runInNewContext(script, {
      document: {
        documentElement: root,
        addEventListener: (type: string, listener: (e: unknown) => void) =>
          listeners.set(type, listener),
      },
      navigator: { userAgent: agent },
      location: { href },
      URL,
    });
    return {
      os: root.dataset.os,
      swapTo: (url) => {
        const next = { dataset: {} as Record<string, string> };
        listeners.get("astro:before-swap")?.({
          newDocument: { documentElement: next },
          to: new URL(url),
        });
        return next.dataset.os;
      },
    };
  };

  it.each(Object.entries(AGENTS))("should mark %s for its readers", (os, agent) => {
    expect(load(agent).os).toBe(os);
  });

  it("should leave an unknown system unmarked", () => {
    expect(load("Mozilla/5.0 (PlayStation)").os).toBeUndefined();
  });

  it("should mark the platform the URL names over the reader's own", () => {
    expect(load(AGENTS.macOS, "https://docs.synnaxlabs.com/?platform=Windows").os).toBe(
      "Windows",
    );
  });

  it("should keep the reader's own platform when the URL names Docker", () => {
    expect(load(AGENTS.Linux, "https://docs.synnaxlabs.com/?platform=Docker").os).toBe(
      "Linux",
    );
  });

  it("should mark the next page before the router swaps it in", () => {
    const page = load(AGENTS.Windows, "https://docs.synnaxlabs.com/?platform=Linux");
    expect(page.swapTo("https://docs.synnaxlabs.com/reference/")).toBe("Windows");
    expect(page.swapTo("https://docs.synnaxlabs.com/reference/?platform=macOS")).toBe(
      "macOS",
    );
  });
});
