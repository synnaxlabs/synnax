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
import { type List } from "@synnaxlabs/lyra/list";
import { array } from "@synnaxlabs/x";
import { act, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/platform/arc";
import { Modals } from "@/platform/modals";
import { reportTaskRunning } from "@/platform/task/testutil";
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

  it("should offer Reload Console", async () => {
    await renderMenu(await createArc());
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should follow whether the running task of the arc drifted", async () => {
    const rck = await client.racks.create({
      name: uniqueName("rack"),
      integrations: ["arc"],
    });
    const a = await createArc();
    const tsk = await client.arcs.updateTask(a.key, { rack: rck.key });
    if (tsk == null) throw new Error("expected a deployment task");
    await reportTaskRunning(client, tsk, "stale");
    // The menu reads only cached tasks, which the list it opens from has fetched.
    await client.arcs.task.retrieve(a.key);
    await renderMenu(a);
    expect(await screen.findByText("Redeploy")).toBeTruthy();
    await act(async () => await reportTaskRunning(client, tsk));
    await waitFor(() => expect(screen.queryByText("Redeploy")).toBeNull());
  });

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
