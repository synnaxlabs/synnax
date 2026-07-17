// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type ontology } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import { client, createSymbolPayload } from "@/feature/schematic/testutil";
import {
  fakePickedFile,
  interceptFilePicker,
  renderHookWithConsole,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

const createSymbolGroup = async (): Promise<group.Group> => {
  const root = await client.schematics.symbols.retrieveGroup();
  return await client.groups.create({
    parent: group.ontologyID(root.key),
    name: uniqueName("symbol_grp"),
  });
};

const childNames = async (id: ontology.ID): Promise<string[]> => {
  const children = await client.ontology.retrieveChildren(id);
  const symbols = await client.schematics.symbols.retrieve({
    keys: children.map((c) => c.key),
  });
  return symbols.map((s) => s.name);
};

describe("Schematic.Symbol.useImport", () => {
  it("imports a picked symbol file into the given group", async () => {
    const grp = await createSymbolGroup();
    const picker = interceptFilePicker();
    const { result } = await renderHookWithConsole(
      () => ({
        run: Schematic.Symbol.useImport(grp.key),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    const name = uniqueName("imported");
    act(() => result.current.run());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([
      fakePickedFile("symbol.json", JSON.stringify(createSymbolPayload(name))),
    ]);
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "success" &&
            st.message === `Successfully imported symbol: ${name}`,
        ),
      ).toBe(true),
    );
    expect(await childNames(group.ontologyID(grp.key))).toContain(name);
  });

  it("surfaces a per-file error when a picked file is not a valid symbol", async () => {
    const grp = await createSymbolGroup();
    const picker = interceptFilePicker();
    const { result } = await renderHookWithConsole(
      () => ({
        run: Schematic.Symbol.useImport(grp.key),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    act(() => result.current.run());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("bad.json", "not json at all")]);
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "error" &&
            st.message === "Failed to import symbol from bad.json",
        ),
      ).toBe(true),
    );
    expect(await childNames(group.ontologyID(grp.key))).toHaveLength(0);
  });

  it("imports nothing when the picker is cancelled", async () => {
    const grp = await createSymbolGroup();
    const picker = interceptFilePicker();
    const { result } = await renderHookWithConsole(
      () => ({
        run: Schematic.Symbol.useImport(grp.key),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    act(() => result.current.run());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.cancel();
    await act(async () => {});
    expect(result.current.notifications.statuses).toHaveLength(0);
    expect(await childNames(group.ontologyID(grp.key))).toHaveLength(0);
  });
});

describe("Schematic.Symbol.useImportGroup", () => {
  const renderImportGroup = async () =>
    await renderHookWithConsole(
      () => ({
        run: Schematic.Symbol.useImportGroup(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );

  it("creates a group and imports every symbol in the manifest", async () => {
    const picker = interceptFilePicker();
    const { result } = await renderImportGroup();
    const groupName = uniqueName("imported_grp");
    const symbolName = uniqueName("sym");
    const manifest = {
      version: 1,
      type: "symbol_group",
      name: groupName,
      symbols: [{ file: "sym1.json", key: "k1", name: symbolName }],
    };
    act(() => result.current.run());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([
      fakePickedFile(
        "manifest.json",
        JSON.stringify(manifest),
        `${groupName}/manifest.json`,
      ),
      fakePickedFile(
        "sym1.json",
        JSON.stringify(createSymbolPayload(symbolName)),
        `${groupName}/sym1.json`,
      ),
    ]);
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "success" &&
            st.message === `Successfully imported 1 symbols into group "${groupName}"`,
        ),
      ).toBe(true),
    );
    const root = await client.schematics.symbols.retrieveGroup();
    const children = await client.ontology.retrieveChildren(group.ontologyID(root.key));
    const groups = await client.groups.retrieve({
      keys: children.filter((c) => c.type === "group").map((c) => c.key),
    });
    const created = groups.find((g) => g.name === groupName);
    if (created == null) throw new Error("imported group not found");
    expect(await childNames(group.ontologyID(created.key))).toContain(symbolName);
  });

  it("raises an error when the directory has no manifest", async () => {
    const picker = interceptFilePicker();
    const { result } = await renderImportGroup();
    act(() => result.current.run());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([
      fakePickedFile("stray.json", "{}", "no_manifest_dir/stray.json"),
    ]);
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "error" && st.message === "Failed to import symbol group",
        ),
      ).toBe(true),
    );
  });

  it("warns when only part of the manifest imports successfully", async () => {
    const picker = interceptFilePicker();
    const { result } = await renderImportGroup();
    const groupName = uniqueName("partial_grp");
    const okName = uniqueName("ok");
    const manifest = {
      version: 1,
      type: "symbol_group",
      name: groupName,
      symbols: [
        { file: "ok.json", key: "k1", name: okName },
        { file: "missing.json", key: "k2", name: "missing" },
      ],
    };
    act(() => result.current.run());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([
      fakePickedFile(
        "manifest.json",
        JSON.stringify(manifest),
        `${groupName}/manifest.json`,
      ),
      fakePickedFile(
        "ok.json",
        JSON.stringify(createSymbolPayload(okName)),
        `${groupName}/ok.json`,
      ),
    ]);
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "warning" &&
            st.message === "Imported 1/2 symbols. Some imports failed.",
        ),
      ).toBe(true),
    );
  });
});
