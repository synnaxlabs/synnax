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
  readFile: vi.fn(),
  readTextFile: vi.fn(),
}));

import { open } from "@tauri-apps/plugin-dialog";
import { readDir, readFile } from "@tauri-apps/plugin-fs";

import { Runtime } from "@/platform/runtime";
import {
  assertDefined,
  fakePickedFile,
  type FilePickerInterceptor,
  interceptFilePicker,
} from "@/testutil";

const openMock = vi.mocked(open);
const readDirMock = vi.mocked(readDir);
const readFileMock = vi.mocked(readFile);

let picker: FilePickerInterceptor;

describe("Runtime files", () => {
  beforeEach(() => {
    mocks.engine = "web";
    for (const m of [openMock, readDirMock, readFileMock]) m.mockReset();
    picker = interceptFilePicker();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("pickFiles (browser)", () => {
    it("should map the selected file into path/readBytes handles", async () => {
      const p = Runtime.pickFiles({ title: "Pick", extension: "json" });
      picker.selectFiles([fakePickedFile("manifest.json", "{}")]);
      const result = await p;
      assertDefined(result);
      expect(result.path).toBe("manifest.json");
      expect(new TextDecoder().decode(await result.readBytes())).toBe("{}");
    });

    it("should return null when no files are selected", async () => {
      const p = Runtime.pickFiles({ title: "Pick", extension: "json" });
      picker.selectFiles([]);
      await expect(p).resolves.toBeNull();
    });

    it("should return null when the picker is cancelled", async () => {
      const p = Runtime.pickFiles({ title: "Pick", extension: "json" });
      picker.cancel();
      await expect(p).resolves.toBeNull();
    });

    it("should build the accept attribute from the extension", async () => {
      const p = Runtime.pickFiles({ title: "Pick", extension: "json", multiple: true });
      const input = picker.lastInput();
      expect(input.accept).toBe(".json");
      expect(input.multiple).toBe(true);
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
      await expect(
        Runtime.pickFiles({ title: "Pick", extension: "json" }),
      ).resolves.toBeNull();
    });

    it("should resolve a single selected path to its basename", async () => {
      openMock.mockResolvedValue("/tmp/data/config.json");
      readFileMock.mockResolvedValue(new Uint8Array([1]));
      const result = await Runtime.pickFiles({ title: "Pick", extension: "json" });
      assertDefined(result);
      expect(result.path).toBe("config.json");
      await result.readBytes();
      expect(readFileMock).toHaveBeenCalledWith("/tmp/data/config.json");
    });

    it("should resolve multiple selected paths", async () => {
      openMock.mockResolvedValue(["/a/one.json", "/b/two.json"]);
      const result = await Runtime.pickFiles({
        title: "Pick",
        extension: "json",
        multiple: true,
      });
      assertDefined(result);
      expect(result.map((f) => f.path)).toEqual(["one.json", "two.json"]);
    });

    it("should return null when the dialog resolves an empty array", async () => {
      openMock.mockResolvedValue([]);
      await expect(
        Runtime.pickFiles({ title: "Pick", extension: "json" }),
      ).resolves.toBeNull();
    });
  });

  describe("pickDirectory (tauri)", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should return null when cancelled", async () => {
      openMock.mockResolvedValue(null);
      await expect(Runtime.pickDirectory({ title: "Pick" })).resolves.toBeNull();
    });

    it("should return null when the dialog resolves an array", async () => {
      openMock.mockResolvedValue(["/a", "/b"]);
      await expect(Runtime.pickDirectory({ title: "Pick" })).resolves.toBeNull();
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
      readFileMock.mockResolvedValue(new Uint8Array([1]));
      const result = await Runtime.pickDirectory({ title: "Pick" });
      assertDefined(result);
      expect(result.name).toBe("project");
      expect(result.files.map((f) => f.path)).toEqual(["a.json", "nested/b.json"]);
      await result.files[0].readBytes();
      expect(readFileMock).toHaveBeenCalledWith("/tmp/project/a.json");
      await result.files[1].readBytes();
      expect(readFileMock).toHaveBeenCalledWith("/tmp/project/nested/b.json");
    });
  });

  describe("pickDirectory (browser)", () => {
    it("should derive the root name and relative paths", async () => {
      const p = Runtime.pickDirectory({ title: "Pick" });
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
      const p = Runtime.pickDirectory({ title: "Pick" });
      picker.selectFiles([]);
      await expect(p).resolves.toBeNull();
    });

    it("should return null when cancelled", async () => {
      const p = Runtime.pickDirectory({ title: "Pick" });
      picker.cancel();
      await expect(p).resolves.toBeNull();
    });
  });

  describe("pickPath", () => {
    it("should reject in the browser without opening a dialog", async () => {
      await expect(Runtime.pickPath({ title: "Pick" })).rejects.toThrow(
        "File paths can only be selected in the Synnax desktop app.",
      );
      expect(openMock).not.toHaveBeenCalled();
    });

    describe("tauri engine", () => {
      beforeEach(() => {
        mocks.engine = "tauri";
      });

      it("should return the chosen absolute path", async () => {
        openMock.mockResolvedValue("/tmp/cert.pem");
        await expect(
          Runtime.pickPath({ title: "Pick", extension: "pem" }),
        ).resolves.toBe("/tmp/cert.pem");
      });

      it("should return null when cancelled", async () => {
        openMock.mockResolvedValue(null);
        await expect(Runtime.pickPath({ title: "Pick" })).resolves.toBeNull();
      });
    });
  });
});
