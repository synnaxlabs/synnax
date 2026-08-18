// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { FS } from "@/platform/fs";
import {
  createJSONFile,
  fakeDataTransfer,
  fakeDirectoryEntry,
  fakeFileEntry,
} from "@/platform/fs/testutil";

describe("captureEntries", () => {
  it("takes only the file items out of a mixed transfer", () => {
    const entry = fakeFileEntry(createJSONFile("a.json", {}));
    expect(FS.captureEntries(fakeDataTransfer([null, entry]))).toEqual([entry]);
  });
});

describe("readEntryFile", () => {
  it("resolves the file the entry points at", async () => {
    const file = createJSONFile("a.json", { key: "a" });
    const entry = fakeFileEntry(file);
    if (!FS.isFileEntry(entry)) throw new Error("expected a file entry");
    await expect(FS.readEntryFile(entry)).resolves.toBe(file);
  });
});

describe("readDirectoryFiles", () => {
  it("records each file's path relative to the dropped directory", async () => {
    const root = fakeDirectoryEntry("root", [
      createJSONFile("top.json", {}),
      fakeDirectoryEntry("nested", [createJSONFile("inner.json", {})]),
    ]);
    if (!FS.isDirectoryEntry(root)) throw new Error("expected a directory entry");
    const files = await FS.readDirectoryFiles(root);
    expect(files.map(({ path }) => path)).toEqual(["top.json", "nested/inner.json"]);
  });

  it("drains a reader that hands entries out in batches", async () => {
    const children = Array.from({ length: 5 }, (_, i) =>
      createJSONFile(`f${i}.json`, {}),
    );
    const root = fakeDirectoryEntry("root", children, 2);
    if (!FS.isDirectoryEntry(root)) throw new Error("expected a directory entry");
    const files = await FS.readDirectoryFiles(root);
    expect(files.map(({ path }) => path)).toEqual([
      "f0.json",
      "f1.json",
      "f2.json",
      "f3.json",
      "f4.json",
    ]);
  });
});
