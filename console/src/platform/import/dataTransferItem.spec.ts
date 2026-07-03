// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import {
  createDataTransferItemContext,
  fakeDataTransferItem,
} from "@/platform/import/testutil";

const jsonFile = (text: string, name: string): File =>
  new File([text], name, { type: "application/json" });

const fileItem = (file: File | null): DataTransferItem =>
  fakeDataTransferItem({ entry: { isFile: true, isDirectory: false }, file });

describe("dataTransferItem", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("throws when the item is not a file", async () => {
    const item = fakeDataTransferItem({ kind: "string" });
    await expect(
      Import.dataTransferItem(item, await createDataTransferItemContext()),
    ).rejects.toThrow("path is null");
  });

  it("ingests a single JSON file, forwarding the parsed data by type", async () => {
    const log = vi.fn();
    const file = jsonFile('{"type":"log","key":"abc"}', "widget.json");
    await Import.dataTransferItem(
      fileItem(file),
      await createDataTransferItemContext({ fileIngesters: { log } }),
    );
    expect(log).toHaveBeenCalledTimes(1);
    expect(log.mock.calls[0][0]).toEqual({ type: "log", key: "abc" });
  });

  it("rejects a non-JSON file", async () => {
    const file = new File(["hello"], "notes.txt", { type: "text/plain" });
    await expect(
      Import.dataTransferItem(fileItem(file), await createDataTransferItemContext()),
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
    const item = fakeDataTransferItem({ entry: directoryEntry });
    const ingestDirectory = vi.fn();
    await Import.dataTransferItem(
      item,
      await createDataTransferItemContext({ ingestDirectory }),
    );
    expect(ingestDirectory).toHaveBeenCalledTimes(1);
    expect(ingestDirectory.mock.calls[0][0]).toEqual("my-directory");
    expect(ingestDirectory.mock.calls[0][1]).toEqual([
      { name: "a.json", data: { type: "log" } },
    ]);
  });
});
