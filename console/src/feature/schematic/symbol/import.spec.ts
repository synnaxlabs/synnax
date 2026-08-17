// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import {
  childNames,
  client,
  createLegacySymbolFile,
} from "@/feature/schematic/testutil";
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

describe("Schematic.Symbol.useImport", () => {
  it("imports a picked symbol file into the given group", async () => {
    const grp = await createSymbolGroup();
    const picker = interceptFilePicker();
    const { result } = await renderHookWithConsole(
      () => ({
        run: Schematic.Symbol.useImport(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    const name = uniqueName("imported");
    act(() => result.current.run(grp.key));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("symbol.json", createLegacySymbolFile(name))]);
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "success" &&
            st.message === "Successfully imported symbol.json",
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
        run: Schematic.Symbol.useImport(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    act(() => result.current.run(grp.key));
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
        run: Schematic.Symbol.useImport(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    act(() => result.current.run(grp.key));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.cancel();
    await act(async () => {});
    expect(result.current.notifications.statuses).toHaveLength(0);
    expect(await childNames(group.ontologyID(grp.key))).toHaveLength(0);
  });
});
