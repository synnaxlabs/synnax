// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { type Import } from "@/import";
import { Layout } from "@/layout";
import { type SliceState, type State, ZERO_SLICE_STATE } from "@/layout/types";
import { Workspace } from "@/workspace";

import { ingest } from "./import";

vi.mock("@synnaxlabs/pluto", async () => {
  const original = await vi.importActual("@synnaxlabs/pluto");
  return { ...original, Access: { updateGranted: () => true } };
});

const makeSlice = (layouts: Record<string, Partial<State>>): SliceState => {
  const fullLayouts: Record<string, State> = {
    main: ZERO_SLICE_STATE.layouts.main,
  };
  for (const [key, partial] of Object.entries(layouts))
    fullLayouts[key] = {
      key,
      name: partial.name ?? key,
      type: partial.type ?? "lineplot",
      location: "mosaic",
      windowKey: "main",
      ...partial,
    } as State;

  const tabs = Object.entries(fullLayouts)
    .filter(([k]) => k !== "main")
    .map(([k, v]) => ({ tabKey: k, name: v.name, closable: true }));

  return {
    ...ZERO_SLICE_STATE,
    layouts: fullLayouts,
    mosaics: {
      main: {
        activeTab: tabs[0]?.tabKey ?? null,
        focused: null,
        root: { key: 1, tabs, selected: tabs[0]?.tabKey },
      },
    },
  };
};

const makeFiles = (
  slice: SliceState,
  components: Record<string, unknown>,
): Import.File[] => {
  const files: Import.File[] = [{ name: Workspace.LAYOUT_FILE_NAME, data: slice }];
  for (const [name, data] of Object.entries(components))
    files.push({ name: `${name}.json`, data });
  return files;
};

const stubContext = () => ({
  client: null,
  fileIngesters: {} as Import.FileIngesters,
  placeLayout: vi.fn(),
  store: { dispatch: vi.fn(), getState: vi.fn() } as unknown as Parameters<
    typeof ingest
  >[2]["store"],
  fluxStore: {} as Parameters<typeof ingest>[2]["fluxStore"],
});

describe("workspace ingest", () => {
  it("should dispatch setActive and setWorkspace with remapped keys", async () => {
    const slice = makeSlice({
      "plot-1": { name: "My Plot", type: "lineplot" },
    });
    const lineplotIngester = vi.fn();
    const ctx = stubContext();
    ctx.fileIngesters = { lineplot: lineplotIngester };
    const files = makeFiles(slice, { "My Plot": { type: "lineplot", key: "plot-1" } });

    await ingest("Test Workspace", files, ctx);

    expect(ctx.store.dispatch).toHaveBeenCalledWith(
      expect.objectContaining({ type: Workspace.setActive.type }),
    );
    expect(ctx.store.dispatch).toHaveBeenCalledWith(
      expect.objectContaining({ type: Layout.setWorkspace.type }),
    );
  });

  it("should call the correct file ingester for each layout entry", async () => {
    const slice = makeSlice({
      "plot-1": { name: "My Plot", type: "lineplot" },
      "sch-1": { name: "My Schematic", type: "schematic" },
    });
    const lineplotIngester = vi.fn();
    const schematicIngester = vi.fn();
    const ctx = stubContext();
    ctx.fileIngesters = {
      lineplot: lineplotIngester,
      schematic: schematicIngester,
    };
    const files = makeFiles(slice, {
      "My Plot": { type: "lineplot", key: "plot-1" },
      "My Schematic": { type: "schematic", key: "sch-1" },
    });

    await ingest("Test Workspace", files, ctx);

    expect(lineplotIngester).toHaveBeenCalledOnce();
    expect(schematicIngester).toHaveBeenCalledOnce();
  });

  it("should throw when LAYOUT.json is missing", async () => {
    const ctx = stubContext();
    const files: Import.File[] = [
      { name: "not-layout.json", data: { type: "lineplot" } },
    ];

    await expect(ingest("Bad Workspace", files, ctx)).rejects.toThrow(
      Workspace.LAYOUT_FILE_NAME,
    );
  });

  it("should throw when a component data file is missing", async () => {
    const slice = makeSlice({
      "plot-1": { name: "My Plot", type: "lineplot" },
    });
    const ctx = stubContext();
    ctx.fileIngesters = { lineplot: vi.fn() };
    const files: Import.File[] = [{ name: Workspace.LAYOUT_FILE_NAME, data: slice }];

    await expect(ingest("Test Workspace", files, ctx)).rejects.toThrow("not found");
  });

  it("should skip layout entries with no matching ingester", async () => {
    const slice = makeSlice({
      "plot-1": { name: "My Plot", type: "lineplot" },
      "unknown-1": { name: "Unknown Thing", type: "unknown_type" },
    });
    const lineplotIngester = vi.fn();
    const ctx = stubContext();
    ctx.fileIngesters = { lineplot: lineplotIngester };
    const files = makeFiles(slice, {
      "My Plot": { type: "lineplot", key: "plot-1" },
      "Unknown Thing": { type: "unknown_type", key: "unknown-1" },
    });

    await ingest("Test Workspace", files, ctx);

    expect(lineplotIngester).toHaveBeenCalledOnce();
  });

  it("should generate new keys different from the original export keys", async () => {
    const slice = makeSlice({
      "plot-1": { name: "My Plot", type: "lineplot" },
    });
    const lineplotIngester = vi.fn();
    const ctx = stubContext();
    ctx.fileIngesters = { lineplot: lineplotIngester };
    const files = makeFiles(slice, { "My Plot": { type: "lineplot", key: "plot-1" } });

    await ingest("Test Workspace", files, ctx);

    const setWorkspaceCall = (
      ctx.store.dispatch as ReturnType<typeof vi.fn>
    ).mock.calls.find(
      ([action]: [{ type: string }]) => action.type === Layout.setWorkspace.type,
    );
    expect(setWorkspaceCall).toBeDefined();
    const newSlice = setWorkspaceCall[0].payload.slice as SliceState;
    const newKeys = Object.keys(newSlice.layouts).filter((k) => k !== "main");
    expect(newKeys).toHaveLength(1);
    expect(newKeys[0]).not.toBe("plot-1");
  });
});
