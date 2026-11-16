// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({
  engine: "web",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/api/path", () => ({
  homeDir: async () => "/Users/synnax",
  join: async (...parts: string[]) => parts.join("/"),
}));
vi.mock("@tauri-apps/plugin-opener", () => ({ revealItemInDir: vi.fn() }));
vi.mock("@tauri-apps/plugin-shell", () => ({ open: vi.fn() }));

import { revealItemInDir } from "@tauri-apps/plugin-opener";
import { open } from "@tauri-apps/plugin-shell";

import { Runtime } from "@/platform/runtime";

const revealMock = vi.mocked(revealItemInDir);
const openMock = vi.mocked(open);

describe("Runtime.openLink", () => {
  beforeEach(() => {
    mocks.engine = "web";
    for (const m of [revealMock, openMock]) m.mockReset();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("a URL", () => {
    it("should open the URL in a new browser tab on the web", async () => {
      const windowOpen = vi.spyOn(window, "open").mockReturnValue(null);
      await Runtime.openLink({ kind: "url", target: "https://example.com/run" });
      expect(windowOpen).toHaveBeenCalledExactlyOnceWith(
        "https://example.com/run",
        "_blank",
        "noopener,noreferrer",
      );
      expect(openMock).not.toHaveBeenCalled();
    });

    it("should hand the URL to the shell plugin in the app", async () => {
      mocks.engine = "tauri";
      await Runtime.openLink({ kind: "url", target: "synnax://range/a1b2" });
      expect(openMock).toHaveBeenCalledExactlyOnceWith("synnax://range/a1b2");
      expect(revealMock).not.toHaveBeenCalled();
    });
  });

  describe("a path", () => {
    it("should show the path in the file manager, and never open it", async () => {
      mocks.engine = "tauri";
      await Runtime.openLink({ kind: "path", target: "/var/log/run.log" });
      expect(revealMock).toHaveBeenCalledExactlyOnceWith("/var/log/run.log");
      expect(openMock).not.toHaveBeenCalled();
    });

    it("should find a path in the home directory of the user", async () => {
      mocks.engine = "tauri";
      await Runtime.openLink({ kind: "path", target: "~/logs/run.log" });
      expect(revealMock).toHaveBeenCalledExactlyOnceWith("/Users/synnax/logs/run.log");
    });

    it("should reject on the web, where a browser cannot show a file", async () => {
      const windowOpen = vi.spyOn(window, "open").mockReturnValue(null);
      await expect(
        Runtime.openLink({ kind: "path", target: "/var/log/run.log" }),
      ).rejects.toThrow("A browser cannot show a file");
      expect(revealMock).not.toHaveBeenCalled();
      expect(windowOpen).not.toHaveBeenCalled();
    });
  });
});
