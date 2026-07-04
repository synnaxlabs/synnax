// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type schematic } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import {
  client,
  createSymbolPayload,
  installFakeDirectoryPicker,
} from "@/feature/schematic/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import {
  captureBrowserDownloads,
  removeFilePickers,
  renderHookWithConsole,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
  removeFilePickers();
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

describe("Schematic.Symbol.useExport", () => {
  it("downloads the symbol as JSON", async () => {
    const downloads = captureBrowserDownloads();
    const name = uniqueName("symbol");
    const root = await client.schematics.symbols.retrieveGroup();
    const symbol = await client.schematics.symbols.create({
      ...createSymbolPayload(name),
      parent: group.ontologyID(root.key),
    });
    const { result } = await renderHookWithConsole(() => Schematic.Symbol.useExport(), {
      client,
    });
    act(() => result.current(symbol.key));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${name}.json`);
    const contents = JSON.parse(
      new TextDecoder().decode(await downloads.blobs[0].arrayBuffer()),
    );
    expect(contents).toMatchObject({ key: symbol.key, name });
    expect(contents.data.svg).toBe(symbol.data.svg);
  });
});

describe("Schematic.Symbol.useExportGroup", () => {
  it("writes a manifest and one file per symbol into the picked directory", async () => {
    const names = [uniqueName("sym_a"), uniqueName("sym_b")];
    const { grp, symbols } = await createSymbolGroup(names);
    const picker = installFakeDirectoryPicker();
    const { result } = await renderHookWithConsole(
      () => ({
        run: Schematic.Symbol.useExportGroup(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    act(() => result.current.run(grp));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "success" &&
            st.message === `Exported 2 symbols to exports/${grp.name}`,
        ),
      ).toBe(true),
    );
    const manifestRaw = picker.files.get("manifest.json");
    if (manifestRaw == null) throw new Error("manifest.json was not written");
    const manifest = Schematic.Symbol.groupManifestZ.parse(JSON.parse(manifestRaw));
    expect(manifest.name).toBe(grp.name);
    expect(manifest.symbols).toHaveLength(2);
    for (const symbol of symbols) {
      const entry = manifest.symbols.find((s) => s.key === symbol.key);
      if (entry == null) throw new Error(`manifest missing symbol ${symbol.name}`);
      const written = picker.files.get(entry.file);
      if (written == null) throw new Error(`symbol file ${entry.file} not written`);
      expect(JSON.parse(written)).toMatchObject({ key: symbol.key });
    }
  });

  it("warns without opening a picker when the group has no symbols", async () => {
    const { grp } = await createSymbolGroup([]);
    const { result } = await renderHookWithConsole(
      () => ({
        run: Schematic.Symbol.useExportGroup(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    act(() => result.current.run(grp));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "warning" &&
            st.message === "No symbols found in this group to export",
        ),
      ).toBe(true),
    );
  });

  it("asks before replacing an existing directory and writes on confirm", async () => {
    const { grp } = await createSymbolGroup([uniqueName("sym")]);
    const picker = installFakeDirectoryPicker({ preExisted: true });
    await renderModalOpener(Schematic.Symbol.useExportGroup, [grp], { client });
    await screen.findByText(`A directory already exists at exports/${grp.name}`);
    fireEvent.click(findButton("Replace"));
    await waitFor(() => expect(picker.files.has("manifest.json")).toBe(true));
  });

  it("writes nothing when the replacement is cancelled", async () => {
    const { grp } = await createSymbolGroup([uniqueName("sym")]);
    const picker = installFakeDirectoryPicker({ preExisted: true });
    await renderModalOpener(Schematic.Symbol.useExportGroup, [grp], { client });
    await screen.findByText(`A directory already exists at exports/${grp.name}`);
    fireEvent.click(findButton("Cancel"));
    await waitFor(() =>
      expect(
        screen.queryByText(`A directory already exists at exports/${grp.name}`),
      ).toBeNull(),
    );
    expect(picker.files.size).toBe(0);
  });
});
