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

vi.mock("@/session/runtime/runtime", () => ({
  get ENGINE() {
    return mocks.engine;
  },
  Drift: class {},
}));

vi.mock("@tauri-apps/plugin-dialog", () => ({ save: vi.fn() }));
vi.mock("@tauri-apps/plugin-fs", () => ({ writeFile: vi.fn() }));

import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";

import { Runtime } from "@/platform/runtime";

const saveMock = vi.mocked(save);
const writeFileMock = vi.mocked(writeFile);

const makeStream = (
  hooks: { onCancel?: () => void } = {},
): ReadableStream<Uint8Array> =>
  new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(new Uint8Array([1, 2, 3]));
      controller.close();
    },
    cancel() {
      hooks.onCancel?.();
    },
  });

describe("Runtime download", () => {
  let createObjectURL: ReturnType<typeof vi.spyOn>;
  let revokeObjectURL: ReturnType<typeof vi.spyOn>;
  let click: ReturnType<typeof vi.spyOn>;
  const clickedAnchors: HTMLAnchorElement[] = [];

  beforeEach(() => {
    mocks.engine = "web";
    saveMock.mockReset();
    writeFileMock.mockReset();
    writeFileMock.mockResolvedValue(undefined);
    clickedAnchors.length = 0;
    delete (window as unknown as Record<string, unknown>).showSaveFilePicker;
    createObjectURL = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:mock-url");
    revokeObjectURL = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
    click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      clickedAnchors.push(this);
    });
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("downloadFromBrowser", () => {
    it("should create an object URL, click a download link, and revoke the URL", () => {
      const blob = new Blob(["hello"], { type: "text/plain" });
      Runtime.downloadFromBrowser(blob, "greeting.txt");
      expect(createObjectURL).toHaveBeenCalledWith(blob);
      expect(click).toHaveBeenCalledTimes(1);
      expect(revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
    });

    it("should set the download filename on the anchor", () => {
      Runtime.downloadFromBrowser(new Blob(["x"]), "report.csv");
      expect(clickedAnchors).toHaveLength(1);
      expect(clickedAnchors[0].download).toBe("report.csv");
    });
  });

  describe("downloadStream", () => {
    describe("showSaveFilePicker (case 1)", () => {
      const installPicker = (name: string) => {
        const writable = new WritableStream<Uint8Array>({ write() {} });
        const fileHandle = {
          name,
          createWritable: vi.fn().mockResolvedValue(writable),
        };
        const showSaveFilePicker = vi.fn().mockResolvedValue(fileHandle);
        (window as unknown as Record<string, unknown>).showSaveFilePicker =
          showSaveFilePicker;
        return { showSaveFilePicker, fileHandle };
      };

      it("should pipe the stream into the chosen file and report status", async () => {
        const { showSaveFilePicker } = installPicker("chosen.csv");
        const addStatus = vi.fn();
        const onDownloadStart = vi.fn();
        await Runtime.downloadStream({
          stream: makeStream(),
          name: "data",
          extension: "csv",
          addStatus,
          onDownloadStart,
        });
        expect(showSaveFilePicker).toHaveBeenCalledWith({
          suggestedName: "data.csv",
        });
        expect(onDownloadStart).toHaveBeenCalledTimes(1);
        expect(addStatus).toHaveBeenNthCalledWith(1, {
          variant: "info",
          message: "Downloading data to chosen.csv",
        });
        expect(addStatus).toHaveBeenNthCalledWith(2, {
          variant: "success",
          message: "Downloaded data to chosen.csv",
        });
      });

      it("should cancel the stream and stay silent when the user aborts", async () => {
        let cancelled = false;
        (window as unknown as Record<string, unknown>).showSaveFilePicker = vi
          .fn()
          .mockRejectedValue(new DOMException("aborted", "AbortError"));
        const addStatus = vi.fn();
        await Runtime.downloadStream({
          stream: makeStream({ onCancel: () => (cancelled = true) }),
          name: "data",
          extension: "csv",
          addStatus,
        });
        expect(cancelled).toBe(true);
        expect(addStatus).not.toHaveBeenCalled();
      });

      it("should rethrow non-abort picker errors", async () => {
        (window as unknown as Record<string, unknown>).showSaveFilePicker = vi
          .fn()
          .mockRejectedValue(new Error("disk on fire"));
        await expect(
          Runtime.downloadStream({
            stream: makeStream(),
            name: "data",
            extension: "csv",
            addStatus: vi.fn(),
          }),
        ).rejects.toThrow("disk on fire");
      });
    });

    describe("tauri stream writer (case 2)", () => {
      beforeEach(() => {
        mocks.engine = "tauri";
      });

      it("should save via dialog and write the stream to disk", async () => {
        saveMock.mockResolvedValue("/home/user/data.csv");
        const addStatus = vi.fn();
        const stream = makeStream();
        await Runtime.downloadStream({
          stream,
          name: "data",
          extension: "csv",
          addStatus,
        });
        expect(saveMock).toHaveBeenCalledWith({
          title: "Download data",
          defaultPath: "data.csv",
        });
        expect(writeFileMock).toHaveBeenCalledWith("/home/user/data.csv", stream);
        expect(addStatus).toHaveBeenNthCalledWith(1, {
          variant: "info",
          message: "Downloading data to /home/user/data.csv",
        });
        expect(addStatus).toHaveBeenNthCalledWith(2, {
          variant: "success",
          message: "Downloaded data to /home/user/data.csv",
        });
      });

      it("should cancel the stream when the save dialog is dismissed", async () => {
        saveMock.mockResolvedValue(null);
        let cancelled = false;
        const addStatus = vi.fn();
        await Runtime.downloadStream({
          stream: makeStream({ onCancel: () => (cancelled = true) }),
          name: "data",
          extension: "csv",
          addStatus,
        });
        expect(cancelled).toBe(true);
        expect(writeFileMock).not.toHaveBeenCalled();
        expect(addStatus).not.toHaveBeenCalled();
      });
    });

    describe("in-memory browser fallback (case 3)", () => {
      it("should buffer the stream to a blob and download it", async () => {
        const addStatus = vi.fn();
        await Runtime.downloadStream({
          stream: makeStream(),
          name: "data",
          extension: "csv",
          addStatus,
        });
        expect(createObjectURL).toHaveBeenCalledTimes(1);
        expect(click).toHaveBeenCalledTimes(1);
        expect(addStatus).toHaveBeenNthCalledWith(1, {
          variant: "info",
          message: "Downloading data to Downloads",
        });
        expect(addStatus).toHaveBeenNthCalledWith(2, {
          variant: "success",
          message: "Downloaded data to Downloads",
        });
      });
    });
  });
});
