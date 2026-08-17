// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type List } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/platform/arc";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper, renderSuspended, uniqueName } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const createArc = async (): Promise<arc.Arc> =>
  await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [] },
  });

const renderMenu = async (a: arc.Arc, as: Client = client): Promise<void> => {
  const getItem = ((keys: arc.Key | arc.Key[]) =>
    Array.isArray(keys) ? [a] : a) as List.GetItem<arc.Key, arc.Arc>;
  const { wrapper } = await createConsoleWrapper({ client: as });
  await renderSuspended(
    <>
      <Arc.ContextMenu
        keys={array.toArray(a.key)}
        getItem={getItem}
        visible
        position={{ x: 0, y: 0 }}
        cursor={{ x: 0, y: 0 }}
      />
      <Modals.Stack />
    </>,
    { wrapper },
  );
};

describe("Arc.ContextMenu", () => {
  it("should offer the write actions to an engineer", async () => {
    await renderMenu(await createArc(), await roles.get("Engineer"));
    expect(await screen.findByText("Edit")).toBeTruthy();
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(await screen.findByText("Delete")).toBeTruthy();
  });

  // The link copy is the read-only affordance, so its presence proves the menu drew.
  it.each(["Viewer", "Operator"] as const)(
    "should withhold the write actions from a %s",
    async (role) => {
      await renderMenu(await createArc(), await roles.get(role));
      expect(await screen.findByText("Copy link")).toBeTruthy();
      expect(screen.queryByText("Edit")).toBeNull();
      expect(screen.queryByText("Rename")).toBeNull();
      expect(screen.queryByText("Delete")).toBeNull();
    },
  );
});
