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
  sep: vi.fn(() => "/"),
  join: vi.fn((...parts: string[]) => Promise.resolve(parts.join("/"))),
}));
vi.mock("@tauri-apps/plugin-dialog", () => ({ open: vi.fn(), save: vi.fn() }));
vi.mock("@tauri-apps/plugin-fs", () => ({
  exists: vi.fn(),
  mkdir: vi.fn(),
  readDir: vi.fn(),
  readTextFile: vi.fn(),
  writeTextFile: vi.fn(),
}));

import { open, save } from "@tauri-apps/plugin-dialog";
import {
  exists,
  mkdir,
  readDir,
  readTextFile,
  writeTextFile,
} from "@tauri-apps/plugin-fs";

import { Runtime } from "@/platform/runtime";
import {
  assertDefined,
  captureBrowserDownloads,
  type CapturedDownloads,
  fakePickedFile,
  type FilePickerInterceptor,
  installDirectoryPicker,
  interceptFilePicker,
  removeFilePickers,
} from "@/testutil";

const openMock = vi.mocked(open);
const saveMock = vi.mocked(save);
const existsMock = vi.mocked(exists);
const mkdirMock = vi.mocked(mkdir);
const readDirMock = vi.mocked(readDir);
const readTextFileMock = vi.mocked(readTextFile);
const writeTextFileMock = vi.mocked(writeTextFile);

let picker: FilePickerInterceptor;

describe("Runtime files", () => {
  beforeEach(() => {
    mocks.engine = "web";
    for (const m of [
      openMock,
      saveMock,
      existsMock,
      mkdirMock,
      readDirMock,
      readTextFileMock,
      writeTextFileMock,
    ])
      m.mockReset();
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

  describe("saveFile (browser)", () => {
    let downloads: CapturedDownloads;
    beforeEach(() => {
      downloads = captureBrowserDownloads();
    });

    it("should download the contents and return the file name", async () => {
      const result = await Runtime.saveFile({
        defaultName: "export.json",
        contents: "{}",
      });
      expect(result).toBe("export.json");
      expect(downloads.blobs).toHaveLength(1);
    });

    it.each([
      ["export.json", "application/json"],
      ["diagram.svg", "image/svg+xml"],
      ["rows.csv", "text/csv"],
      ["doc.xml", "application/xml"],
      ["notes.txt", "text/plain"],
      ["archive.bin", "application/octet-stream"],
      ["noext", "application/octet-stream"],
    ])("should infer the mime type of %s as %s", async (name, mime) => {
      await Runtime.saveFile({ defaultName: name, contents: "x" });
      expect(downloads.blobs[0].type).toBe(mime);
    });
  });

  describe("saveFile (tauri)", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should write contents to the chosen path and return it", async () => {
      saveMock.mockResolvedValue("/tmp/export.json");
      const result = await Runtime.saveFile({
        defaultName: "export.json",
        contents: "{}",
      });
      expect(result).toBe("/tmp/export.json");
      expect(writeTextFileMock).toHaveBeenCalledWith("/tmp/export.json", "{}");
    });

    it("should return null when the save dialog is cancelled", async () => {
      saveMock.mockResolvedValue(null);
      const result = await Runtime.saveFile({
        defaultName: "export.json",
        contents: "{}",
      });
      expect(result).toBeNull();
      expect(writeTextFileMock).not.toHaveBeenCalled();
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

    it("should collect only the files in the directory", async () => {
      openMock.mockResolvedValue("/tmp/project");
      readDirMock.mockResolvedValue([
        { name: "a.json", isFile: true, isDirectory: false, isSymlink: false },
        { name: "nested", isFile: false, isDirectory: true, isSymlink: false },
        { name: "b.json", isFile: true, isDirectory: false, isSymlink: false },
      ]);
      readTextFileMock.mockResolvedValue("data");
      const result = await Runtime.pickDirectory();
      assertDefined(result);
      expect(result.name).toBe("project");
      expect(result.files.map((f) => f.name)).toEqual(["a.json", "b.json"]);
      await result.files[0].read();
      expect(readTextFileMock).toHaveBeenCalledWith("/tmp/project/a.json");
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

  describe("pickWritableDirectory (tauri)", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should return null when cancelled", async () => {
      openMock.mockResolvedValue(null);
      await expect(
        Runtime.pickWritableDirectory({ subdirectory: "out" }),
      ).resolves.toBeNull();
    });

    it("should report preExisted and write into the subdirectory once ensured", async () => {
      openMock.mockResolvedValue("/tmp/parent");
      existsMock.mockResolvedValue(true);
      mkdirMock.mockResolvedValue(undefined);
      writeTextFileMock.mockResolvedValue(undefined);
      const dir = await Runtime.pickWritableDirectory({ subdirectory: "out" });
      assertDefined(dir);
      expect(dir.displayPath).toBe("/tmp/parent/out");
      expect(dir.preExisted).toBe(true);
      await dir.writeText("a.json", "{}");
      await dir.writeText("b.json", "[]");
      expect(mkdirMock).toHaveBeenCalledTimes(1);
      expect(mkdirMock).toHaveBeenCalledWith("/tmp/parent/out", {
        recursive: true,
      });
      expect(writeTextFileMock).toHaveBeenCalledWith("/tmp/parent/out/a.json", "{}");
      expect(writeTextFileMock).toHaveBeenCalledWith("/tmp/parent/out/b.json", "[]");
    });

    it("should report preExisted false for a fresh subdirectory", async () => {
      openMock.mockResolvedValue("/tmp/parent");
      existsMock.mockResolvedValue(false);
      const dir = await Runtime.pickWritableDirectory({ subdirectory: "out" });
      assertDefined(dir);
      expect(dir.preExisted).toBe(false);
    });
  });

  describe("pickWritableDirectory (browser)", () => {
    afterEach(() => {
      removeFilePickers();
    });

    it("should throw a clear error when the browser lacks showDirectoryPicker", async () => {
      removeFilePickers();
      await expect(
        Runtime.pickWritableDirectory({ subdirectory: "out" }),
      ).rejects.toThrow(/does not support/);
    });

    it("should return null when the directory picker is aborted", async () => {
      installDirectoryPicker(
        vi.fn().mockRejectedValue(new DOMException("aborted", "AbortError")),
      );
      await expect(
        Runtime.pickWritableDirectory({ subdirectory: "out" }),
      ).resolves.toBeNull();
    });

    it("should write into an existing subdirectory", async () => {
      const write = vi.fn().mockResolvedValue(undefined);
      const close = vi.fn().mockResolvedValue(undefined);
      const fileHandle = {
        createWritable: vi.fn().mockResolvedValue({ write, close }),
      };
      const subHandle = {
        getFileHandle: vi.fn().mockResolvedValue(fileHandle),
      };
      const root = {
        name: "Downloads",
        getDirectoryHandle: vi.fn().mockResolvedValue(subHandle),
      };
      installDirectoryPicker(vi.fn().mockResolvedValue(root));
      const dir = await Runtime.pickWritableDirectory({ subdirectory: "out" });
      assertDefined(dir);
      expect(dir.displayPath).toBe("Downloads/out");
      expect(dir.preExisted).toBe(true);
      await dir.writeText("a.json", "{}");
      expect(subHandle.getFileHandle).toHaveBeenCalledWith("a.json", {
        create: true,
      });
      expect(write).toHaveBeenCalledWith("{}");
      expect(close).toHaveBeenCalledTimes(1);
    });

    it("should create the subdirectory lazily when it does not yet exist", async () => {
      const notFound = new DOMException("missing", "NotFoundError");
      const write = vi.fn().mockResolvedValue(undefined);
      const close = vi.fn().mockResolvedValue(undefined);
      const fileHandle = {
        createWritable: vi.fn().mockResolvedValue({ write, close }),
      };
      const createdSub = {
        getFileHandle: vi.fn().mockResolvedValue(fileHandle),
      };
      const getDirectoryHandle = vi
        .fn()
        .mockImplementationOnce(() => Promise.reject(notFound))
        .mockImplementationOnce(() => Promise.resolve(createdSub));
      const root = { name: "Downloads", getDirectoryHandle };
      installDirectoryPicker(vi.fn().mockResolvedValue(root));
      const dir = await Runtime.pickWritableDirectory({ subdirectory: "out" });
      assertDefined(dir);
      expect(dir.preExisted).toBe(false);
      await dir.writeText("a.json", "{}");
      expect(getDirectoryHandle).toHaveBeenNthCalledWith(2, "out", {
        create: true,
      });
      expect(write).toHaveBeenCalledWith("{}");
    });

    it("should rethrow non-abort errors from the directory picker", async () => {
      installDirectoryPicker(
        vi.fn().mockRejectedValue(new DOMException("denied", "SecurityError")),
      );
      await expect(
        Runtime.pickWritableDirectory({ subdirectory: "out" }),
      ).rejects.toThrow();
    });

    it("should rethrow unexpected errors from getDirectoryHandle", async () => {
      const root = {
        name: "Downloads",
        getDirectoryHandle: vi
          .fn()
          .mockRejectedValue(new DOMException("boom", "SecurityError")),
      };
      installDirectoryPicker(vi.fn().mockResolvedValue(root));
      await expect(
        Runtime.pickWritableDirectory({ subdirectory: "out" }),
      ).rejects.toThrow();
    });
  });
});
