// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import { Session } from "@/session";

const makeStore = () =>
  configureStore({
    reducer: { [Session.Project.SLICE_NAME]: Session.Project.reducer },
    preloadedState: {
      [Session.Project.SLICE_NAME]: {
        ...Session.Project.ZERO_SLICE_STATE,
        selected: "project-1",
      },
    },
  });

const makeContext = (over: Partial<Parameters<typeof Import.dataTransferItem>[1]>) =>
  ({
    client: null,
    fileIngesters: {},
    ingestDirectory: vi.fn(),
    layout: {},
    placeLayout: vi.fn(),
    store: makeStore(),
    fluxStore: {},
    ...over,
  }) as unknown as Parameters<typeof Import.dataTransferItem>[1];

// jsdom's File/Blob does not implement arrayBuffer, which dataTransferItem reads to
// decode file contents. Back the instance with the real bytes so decoding works.
const withArrayBuffer = <T extends { arrayBuffer?: () => Promise<ArrayBuffer> }>(
  file: T,
  text: string,
): T => {
  Object.defineProperty(file, "arrayBuffer", {
    value: async () => new TextEncoder().encode(text).buffer,
  });
  return file;
};

const jsonFile = (text: string, name: string): File =>
  withArrayBuffer(new File([text], name, { type: "application/json" }), text);

const fileItem = (file: File | null): DataTransferItem =>
  ({
    kind: "file",
    webkitGetAsEntry: () => ({ isFile: true, isDirectory: false }),
    getAsFile: () => file,
  }) as unknown as DataTransferItem;

describe("dataTransferItem", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("throws when the item is not a file", async () => {
    const item = { kind: "string" } as unknown as DataTransferItem;
    await expect(Import.dataTransferItem(item, makeContext({}))).rejects.toThrow(
      "path is null",
    );
  });

  it("ingests a single JSON file, forwarding the parsed data by type", async () => {
    const log = vi.fn();
    const file = jsonFile('{"type":"log","key":"abc"}', "widget.json");
    await Import.dataTransferItem(
      fileItem(file),
      makeContext({ fileIngesters: { log } }),
    );
    expect(log).toHaveBeenCalledTimes(1);
    expect(log.mock.calls[0][0]).toEqual({ type: "log", key: "abc" });
  });

  it("rejects a non-JSON file", async () => {
    const file = new File(["hello"], "notes.txt", { type: "text/plain" });
    await expect(
      Import.dataTransferItem(fileItem(file), makeContext({})),
    ).rejects.toThrow("not a JSON file");
  });

  it("reads a directory transfer and hands its parsed files to the directory ingester", async () => {
    const fileEntry = {
      isFile: true,
      isDirectory: false,
      file: (resolve: (f: File) => void) =>
        resolve(jsonFile('{"type":"log"}', "a.json")),
    };
    let readCount = 0;
    const directoryEntry = {
      isFile: false,
      isDirectory: true,
      name: "my-directory",
      createReader: () => ({
        readEntries: (resolve: (entries: unknown[]) => void) =>
          resolve(readCount++ === 0 ? [fileEntry] : []),
      }),
    };
    const item = {
      kind: "file",
      webkitGetAsEntry: () => directoryEntry,
    } as unknown as DataTransferItem;
    const ingestDirectory = vi.fn();
    await Import.dataTransferItem(item, makeContext({ ingestDirectory }));
    expect(ingestDirectory).toHaveBeenCalledTimes(1);
    expect(ingestDirectory.mock.calls[0][0]).toEqual("my-directory");
    expect(ingestDirectory.mock.calls[0][1]).toEqual([
      { name: "a.json", data: { type: "log" } },
    ]);
  });
});
