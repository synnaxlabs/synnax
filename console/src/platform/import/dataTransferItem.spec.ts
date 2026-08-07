// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import {
  createDataTransferItemContext,
  fakeDataTransferItem,
} from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { Session } from "@/session";
import { uniqueName } from "@/testutil";

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

  it("streams a single JSON file to the Core", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const original = await client.logs.create(proj.key, { name: uniqueName("log") });
    const stream = await client.imex.export(log.ontologyID(original.key), {
      encoding: "JSON",
    });
    const data = await new Response(stream).text();
    const openTab = vi.fn<Panel.OpenTab>();
    const ctx = await createDataTransferItemContext({ client, openTab });
    ctx.store.dispatch(Session.Project.select(proj.key));
    await Import.dataTransferItem(fileItem(jsonFile(data, "widget.json")), ctx);
    expect(openTab).toHaveBeenCalledTimes(1);
    const [tab] = openTab.mock.calls[0];
    if (tab.variant !== "resource" || typeof tab.resource === "string")
      throw new Error("expected a resource tab");
    const created = await client.logs.retrieve({ key: tab.resource.key });
    expect(created.name).toBe(original.name);
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
