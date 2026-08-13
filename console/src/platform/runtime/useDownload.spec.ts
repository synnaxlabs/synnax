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

vi.mock("@tauri-apps/plugin-dialog", () => ({ save: vi.fn() }));
vi.mock("@tauri-apps/plugin-fs", () => ({ writeFile: vi.fn() }));

import { Status } from "@synnaxlabs/pluto";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";
import { act } from "@testing-library/react";

import { Runtime } from "@/platform/runtime";
import {
  captureBrowserDownloads,
  type CapturedDownloads,
  fakeSaveFileHandle,
  installSaveFilePicker,
  MOCK_OBJECT_URL,
  removeFilePickers,
  renderHookWithConsole,
} from "@/testutil";

const saveMock = vi.mocked(save);
const writeFileMock = vi.mocked(writeFile);

const createStream = (
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

const renderDownload = async () =>
  await renderHookWithConsole(() => ({
    download: Runtime.useDownload(),
    notifications: Status.useNotifications(),
  }));

describe("Runtime.useDownload", () => {
  let downloads: CapturedDownloads;

  beforeEach(() => {
    mocks.engine = "web";
    saveMock.mockReset();
    writeFileMock.mockReset();
    writeFileMock.mockResolvedValue(undefined);
    removeFilePickers();
    downloads = captureBrowserDownloads();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("showSaveFilePicker (case 1)", () => {
    it("should pipe the stream into the chosen file and report status", async () => {
      const showSaveFilePicker = vi
        .fn()
        .mockResolvedValue(fakeSaveFileHandle("chosen.csv"));
      installSaveFilePicker(showSaveFilePicker);
      const onDownloadStart = vi.fn();
      const { result } = await renderDownload();
      await act(async () => {
        await result.current.download({
          stream: createStream(),
          name: "data",
          extension: "csv",
          onDownloadStart,
        });
      });
      expect(showSaveFilePicker).toHaveBeenCalledWith({
        suggestedName: "data.csv",
      });
      expect(onDownloadStart).toHaveBeenCalledTimes(1);
      const { statuses } = result.current.notifications;
      expect(statuses.map(({ variant, message }) => ({ variant, message }))).toEqual(
        expect.arrayContaining([
          { variant: "info", message: "Downloading data to chosen.csv" },
          { variant: "success", message: "Downloaded data to chosen.csv" },
        ]),
      );
    });

    it("should cancel the stream and stay silent when the user aborts", async () => {
      let cancelled = false;
      installSaveFilePicker(
        vi.fn().mockRejectedValue(new DOMException("aborted", "AbortError")),
      );
      const { result } = await renderDownload();
      await act(async () => {
        await result.current.download({
          stream: createStream({ onCancel: () => (cancelled = true) }),
          name: "data",
          extension: "csv",
        });
      });
      expect(cancelled).toBe(true);
      expect(result.current.notifications.statuses).toHaveLength(0);
    });

    it("should rethrow non-abort picker errors", async () => {
      installSaveFilePicker(vi.fn().mockRejectedValue(new Error("disk on fire")));
      const { result } = await renderDownload();
      await act(async () => {
        await expect(
          result.current.download({
            stream: createStream(),
            name: "data",
            extension: "csv",
          }),
        ).rejects.toThrow("disk on fire");
      });
    });
  });

  describe("tauri stream writer (case 2)", () => {
    it("should save via dialog and write the stream to disk", async () => {
      const { result } = await renderDownload();
      mocks.engine = "tauri";
      saveMock.mockResolvedValue("/home/user/data.csv");
      const stream = createStream();
      const filters = [{ name: "CSV", extensions: ["csv"] }];
      await act(async () => {
        await result.current.download({
          stream,
          name: "data",
          extension: "csv",
          filters,
        });
      });
      expect(saveMock).toHaveBeenCalledWith({
        title: "Download data",
        defaultPath: "data.csv",
        filters,
      });
      expect(writeFileMock).toHaveBeenCalledWith("/home/user/data.csv", stream);
      const { statuses } = result.current.notifications;
      expect(statuses.map(({ variant, message }) => ({ variant, message }))).toEqual(
        expect.arrayContaining([
          { variant: "info", message: "Downloading data to /home/user/data.csv" },
          { variant: "success", message: "Downloaded data to /home/user/data.csv" },
        ]),
      );
    });

    it("should title the dialog with the raw name and sanitize the file name", async () => {
      const { result } = await renderDownload();
      mocks.engine = "tauri";
      saveMock.mockResolvedValue("/home/user/new_ group_.zip");
      await act(async () => {
        await result.current.download({
          stream: createStream(),
          name: "new: group?",
          extension: "zip",
        });
      });
      expect(saveMock).toHaveBeenCalledWith({
        title: "Download new: group?",
        defaultPath: "new_ group_.zip",
        filters: undefined,
      });
    });

    it("should cancel the stream when the save dialog is dismissed", async () => {
      const { result } = await renderDownload();
      mocks.engine = "tauri";
      saveMock.mockResolvedValue(null);
      let cancelled = false;
      await act(async () => {
        await result.current.download({
          stream: createStream({ onCancel: () => (cancelled = true) }),
          name: "data",
          extension: "csv",
        });
      });
      expect(cancelled).toBe(true);
      expect(writeFileMock).not.toHaveBeenCalled();
      expect(result.current.notifications.statuses).toHaveLength(0);
    });
  });

  describe("in-memory browser fallback (case 3)", () => {
    it("should buffer the stream to a blob and download it", async () => {
      const { result } = await renderDownload();
      await act(async () => {
        await result.current.download({
          stream: createStream(),
          name: "data",
          extension: "csv",
        });
      });
      expect(downloads.blobs).toHaveLength(1);
      expect(downloads.anchors).toHaveLength(1);
      expect(downloads.anchors[0].download).toBe("data.csv");
      expect(downloads.revoked).toEqual([MOCK_OBJECT_URL]);
      const { statuses } = result.current.notifications;
      expect(statuses.map(({ variant, message }) => ({ variant, message }))).toEqual(
        expect.arrayContaining([
          { variant: "info", message: "Downloading data to Downloads" },
          { variant: "success", message: "Downloaded data to Downloads" },
        ]),
      );
    });
  });
});
