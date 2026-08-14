// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError, type schematic } from "@synnaxlabs/client";
import { errors } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { client, createSymbolPayload } from "@/feature/schematic/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { uniqueName } from "@/testutil";

const createSymbolGroup = async (
  symbolNames: string[],
): Promise<{ grp: group.Group; symbols: schematic.symbol.Symbol[] }> => {
  const root = await client.schematics.symbols.retrieveGroup();
  const grp = await client.groups.create({
    parent: group.ontologyID(root.key),
    name: uniqueName("delete_grp"),
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

const groupExists = async (key: group.Key): Promise<boolean> => {
  try {
    await client.ontology.retrieve(group.ontologyID(key));
    return true;
  } catch (e) {
    if (NotFoundError.matches(e)) return false;
    throw errors.fromUnknown(e);
  }
};

describe("Schematic.Symbol.useDeleteGroup", () => {
  it("deletes the group and its symbols after confirmation", async () => {
    const { grp, symbols } = await createSymbolGroup([
      uniqueName("sym_a"),
      uniqueName("sym_b"),
    ]);
    await renderModalOpener(Schematic.Symbol.useDeleteGroup, [grp], { client });
    await screen.findByText(`Are you sure you want to delete ${grp.name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => expect(await groupExists(grp.key)).toBe(false));
    await expect(
      client.schematics.symbols.retrieve({ keys: symbols.map((s) => s.key) }),
    ).rejects.toSatisfy((e) => NotFoundError.matches(e));
  });

  it("keeps the group when the deletion is cancelled", async () => {
    const { grp } = await createSymbolGroup([uniqueName("sym")]);
    const { grp: control } = await createSymbolGroup([uniqueName("sym")]);
    const target = { current: grp };
    const handle = await renderModalOpener(
      () => {
        const del = Schematic.Symbol.useDeleteGroup();
        return () => del(target.current);
      },
      [],
      { client },
    );
    await screen.findByText(`Are you sure you want to delete ${grp.name}?`);
    fireEvent.click(findButton("Cancel"));
    await waitFor(() =>
      expect(
        screen.queryByText(`Are you sure you want to delete ${grp.name}?`),
      ).toBeNull(),
    );
    target.current = control;
    handle.reopen();
    await screen.findByText(`Are you sure you want to delete ${control.name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => expect(await groupExists(control.key)).toBe(false));
    expect(await groupExists(grp.key)).toBe(true);
  });
});
