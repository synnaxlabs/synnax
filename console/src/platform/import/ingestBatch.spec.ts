// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import { type Panel } from "@/platform/panel";

interface Item {
  name: string;
}

const createItem = (name: string): Item => ({ name });

const createID = () => log.ontologyID(uuid.create());

describe("Import.ingestBatch", () => {
  it("opens the created resources as one batch of tabs in item order", async () => {
    const [first, second] = [createID(), createID()];
    const openTabs = vi.fn<Panel.OpenTabs>();
    await Import.ingestBatch({
      items: [createItem("a.json"), createItem("b.json")],
      ingest: async ({ name }) => (name === "a.json" ? first : second),
      handleError: vi.fn(),
      openTabs,
    });
    expect(openTabs).toHaveBeenCalledTimes(1);
    expect(openTabs.mock.calls[0][0]).toEqual([
      { variant: "resource", resource: first },
      { variant: "resource", resource: second },
    ]);
  });

  // A dropped directory imports a whole project, which brings its own panels, so it
  // creates no resource for the batch to open.
  it("opens no tab for an item that created no resource", async () => {
    const id = createID();
    const openTabs = vi.fn<Panel.OpenTabs>();
    await Import.ingestBatch({
      items: [createItem("my-project"), createItem("a.json")],
      ingest: async ({ name }) => (name === "a.json" ? id : undefined),
      handleError: vi.fn(),
      openTabs,
    });
    expect(openTabs.mock.calls[0][0]).toEqual([{ variant: "resource", resource: id }]);
  });

  it("reports a failed item by name and still opens tabs for the rest", async () => {
    const id = createID();
    const boom = new Error("not a JSON file");
    const handleError = vi.fn();
    const openTabs = vi.fn<Panel.OpenTabs>();
    await Import.ingestBatch({
      items: [createItem("notes.txt"), createItem("a.json")],
      ingest: async ({ name }) => {
        if (name === "notes.txt") throw boom;
        return id;
      },
      handleError,
      openTabs,
    });
    expect(handleError).toHaveBeenCalledWith(boom, "Failed to import notes.txt");
    expect(openTabs.mock.calls[0][0]).toEqual([{ variant: "resource", resource: id }]);
  });

  it("ingests concurrently and orders tabs by item, not by finish order", async () => {
    let release = (): void => {};
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    const [slow, fast] = [createID(), createID()];
    const finished: string[] = [];
    const openTabs = vi.fn<Panel.OpenTabs>();
    const batch = Import.ingestBatch({
      items: [createItem("slow.json"), createItem("fast.json")],
      ingest: async ({ name }) => {
        const isSlow = name === "slow.json";
        if (isSlow) await held;
        finished.push(name);
        return isSlow ? slow : fast;
      },
      handleError: vi.fn(),
      openTabs,
    });
    await waitFor(() => expect(finished).toEqual(["fast.json"]));
    expect(openTabs).not.toHaveBeenCalled();
    release();
    await batch;
    expect(openTabs.mock.calls[0][0]).toEqual([
      { variant: "resource", resource: slow },
      { variant: "resource", resource: fast },
    ]);
  });
});
