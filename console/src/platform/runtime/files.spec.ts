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

vi.mock("@tauri-apps/api/path", () => ({ sep: vi.fn(() => "/") }));
vi.mock("@tauri-apps/plugin-dialog", () => ({ open: vi.fn() }));
vi.mock("@tauri-apps/plugin-fs", () => ({
  readDir: vi.fn(),
  readTextFile: vi.fn(),
}));

import { open } from "@tauri-apps/plugin-dialog";
import { readDir, readTextFile } from "@tauri-apps/plugin-fs";

import { Runtime } from "@/platform/runtime";
import {
  assertDefined,
  fakePickedFile,
  type FilePickerInterceptor,
  interceptFilePicker,
} from "@/testutil";

const openMock = vi.mocked(open);
const readDirMock = vi.mocked(readDir);
const readTextFileMock = vi.mocked(readTextFile);

let picker: FilePickerInterceptor;

describe("Runtime files", () => {
  beforeEach(() => {
    mocks.engine = "web";
    for (const m of [openMock, readDirMock, readTextFileMock]) m.mockReset();
    picker = interceptFilePicker();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("pickFiles (browser)", () => {
    it("should map selected files into name/path/read handles", async () => {
      const p = Runtime.pickFiles({});
      picker.selectFiles([fakePickedFile("manifest.json", "{}")]);
      const result = await p;
      assertDefined(result);
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe("manifest.json");
      expect(result[0].path).toBe("manifest.json");
      await expect(result[0].read()).resolves.toBe("{}");
    });

    it("should return null when no files are selected", async () => {
      const p = Runtime.pickFiles({});
      picker.selectFiles([]);
      await expect(p).resolves.toBeNull();
    });

    it("should return null when the picker is cancelled", async () => {
      const p = Runtime.pickFiles({});
      picker.cancel();
      await expect(p).resolves.toBeNull();
    });

    it("should build the accept attribute from filters", async () => {
      const p = Runtime.pickFiles({
        filters: [{ name: "data", extensions: ["json", "csv"] }],
        multiple: true,
      });
      const input = picker.lastInput();
      expect(input.accept).toBe(".json,.csv");
      expect(input.multiple).toBe(true);
      picker.cancel();
      await p;
    });

    it("should leave accept unset when there are no filters", async () => {
      const p = Runtime.pickFiles({});
      expect(picker.lastInput().accept).toBe("");
      picker.cancel();
      await p;
    });
  });

  describe("pickFiles (tauri)", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should return null when the dialog is cancelled", async () => {
      openMock.mockResolvedValue(null);
      await expect(Runtime.pickFiles({})).resolves.toBeNull();
    });

    it("should resolve a single selected path to its basename", async () => {
      openMock.mockResolvedValue("/tmp/data/config.json");
      readTextFileMock.mockResolvedValue("{}");
      const result = await Runtime.pickFiles({ title: "Pick" });
      assertDefined(result);
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe("config.json");
      expect(result[0].path).toBe("config.json");
      await result[0].read();
      expect(readTextFileMock).toHaveBeenCalledWith("/tmp/data/config.json");
    });

    it("should resolve multiple selected paths", async () => {
      openMock.mockResolvedValue(["/a/one.json", "/b/two.json"]);
      const result = await Runtime.pickFiles({ multiple: true });
      assertDefined(result);
      expect(result.map((f) => f.name)).toEqual(["one.json", "two.json"]);
    });

    it("should return null when the dialog resolves an empty array", async () => {
      openMock.mockResolvedValue([]);
      await expect(Runtime.pickFiles({})).resolves.toBeNull();
    });
  });

  describe("pickDirectory (tauri)", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should return null when cancelled", async () => {
      openMock.mockResolvedValue(null);
      await expect(Runtime.pickDirectory()).resolves.toBeNull();
    });

    it("should return null when the dialog resolves an array", async () => {
      openMock.mockResolvedValue(["/a", "/b"]);
      await expect(Runtime.pickDirectory()).resolves.toBeNull();
    });

    it("should collect files recursively with relative paths", async () => {
      openMock.mockResolvedValue("/tmp/project");
      readDirMock.mockImplementation(async (path) =>
        path === "/tmp/project"
          ? [
              { name: "a.json", isFile: true, isDirectory: false, isSymlink: false },
              { name: "nested", isFile: false, isDirectory: true, isSymlink: false },
            ]
          : [{ name: "b.json", isFile: true, isDirectory: false, isSymlink: false }],
      );
      readTextFileMock.mockResolvedValue("data");
      const result = await Runtime.pickDirectory();
      assertDefined(result);
      expect(result.name).toBe("project");
      expect(result.files.map((f) => f.path)).toEqual(["a.json", "nested/b.json"]);
      await result.files[0].read();
      expect(readTextFileMock).toHaveBeenCalledWith("/tmp/project/a.json");
      await result.files[1].read();
      expect(readTextFileMock).toHaveBeenCalledWith("/tmp/project/nested/b.json");
    });
  });

  describe("pickDirectory (browser)", () => {
    it("should derive the root name and relative paths", async () => {
      const p = Runtime.pickDirectory();
      expect(picker.lastInput().webkitdirectory).toBe(true);
      picker.selectFiles([
        fakePickedFile("foo.json", "1", "myroot/foo.json"),
        fakePickedFile("bar.json", "2", "myroot/sub/bar.json"),
      ]);
      const result = await p;
      assertDefined(result);
      expect(result.name).toBe("myroot");
      expect(result.files.map((f) => f.path)).toEqual(["foo.json", "sub/bar.json"]);
    });

    it("should return null when nothing is selected", async () => {
      const p = Runtime.pickDirectory();
      picker.selectFiles([]);
      await expect(p).resolves.toBeNull();
    });

    it("should return null when cancelled", async () => {
      const p = Runtime.pickDirectory();
      picker.cancel();
      await expect(p).resolves.toBeNull();
    });
  });
});
