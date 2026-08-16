// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, schematic } from "@synnaxlabs/client";
import { act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { client, createSymbolPayload } from "@/feature/schematic/testutil";
import { Export } from "@/platform/export";
import {
  captureBrowserDownloads,
  removeSaveFilePicker,
  renderHookWithConsole,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
  removeSaveFilePicker();
});

const createSymbolGroup = async (
  symbolNames: string[],
): Promise<{ grp: group.Group; symbols: schematic.symbol.Symbol[] }> => {
  const root = await client.schematics.symbols.retrieveGroup();
  const grp = await client.groups.create({
    parent: group.ontologyID(root.key),
    name: uniqueName("export_grp"),
  });
  const symbols: schematic.symbol.Symbol[] = [];
  for (const name of symbolNames)
    symbols.push(
      await client.schematics.symbols.create({
        ...createSymbolPayload(name),
        parent: group.ontologyID(grp.key),
      }),
    );
  return { grp, symbols };
};

describe("exporting a symbol", () => {
  it("downloads the symbol as JSON", async () => {
    const downloads = captureBrowserDownloads();
    const name = uniqueName("symbol");
    const root = await client.schematics.symbols.retrieveGroup();
    const symbol = await client.schematics.symbols.create({
      ...createSymbolPayload(name),
      parent: group.ontologyID(root.key),
    });
    const { result } = await renderHookWithConsole(() => Export.useResource(), {
      client,
    });
    act(() => result.current({ id: schematic.symbol.ontologyID(symbol.key), name }));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${name}.json`);
    const contents = JSON.parse(
      new TextDecoder().decode(await downloads.blobs[0].arrayBuffer()),
    );
    expect(contents).toMatchObject({ name, type: "schematic_symbol" });
    expect(contents.data.svg).toBe(symbol.data.svg);
  });
});

describe("exporting a symbol group", () => {
  it("downloads the group as a zip named after the group", async () => {
    const downloads = captureBrowserDownloads();
    const names = [uniqueName("sym_a"), uniqueName("sym_b")];
    const { grp } = await createSymbolGroup(names);
    const { result } = await renderHookWithConsole(() => Export.use(), { client });
    act(() =>
      result.current({
        stream: (c) => c.schematics.symbols.exportGroup(grp.key, { encoding: "JSON" }),
        name: grp.name,
        extension: "zip",
      }),
    );
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${grp.name}.zip`);
    // Zip entry names are stored uncompressed, so the archive names its own files.
    const archive = new TextDecoder().decode(await downloads.blobs[0].arrayBuffer());
    expect(archive.startsWith("PK")).toBe(true);
    expect(archive).toContain("manifest.json");
    names.forEach((name) => expect(archive).toContain(`${name}.json`));
  });
});
