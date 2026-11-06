// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Table } from "@/feature/table";
import { client, project } from "@/feature/table/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { renderVisible, select } from "@/platform/selector/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import { assertDefined, createConsoleWrapper, resolveFocusedTab } from "@/testutil";

const roles = new RoleClients(client);

describe("table/Selectable", () => {
  it("creates a table in the active project and opens its tab when selected", async () => {
    const proj = await client.projects.retrieve(await project());
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    const Selectable = Table.SELECTABLES[0];
    expect(Selectable.type).toBe(table.TYPE_ONTOLOGY_ID.type);
    select(Selectable, wrapper);
    const tab = await resolveFocusedTab(store, client, (t) => t.variant === "resource");
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(table.TYPE_ONTOLOGY_ID.type);
    const key = tab.resource.key;
    const created = await client.tables.retrieve(key);
    expect(created.name).toBe("Table");
    await waitFor(() =>
      expect(
        documentIn(store.getState()[Session.Table.SLICE_NAME], key)?.editable,
      ).toBe(true),
    );
  });
});

describe("table/Selectable permissions", () => {
  const findSelectable = () => {
    const Selectable = Table.SELECTABLES.find(
      (s) => s.type === table.TYPE_ONTOLOGY_ID.type,
    );
    assertDefined(Selectable, "no selectable registered for table");
    return Selectable;
  };

  it("should be visible to an engineer", async () => {
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Engineer"),
    });
    const { result } = renderVisible(findSelectable(), table.TYPE_ONTOLOGY_ID, wrapper);
    await waitFor(() => expect(result.current.visible).toBe(true));
  });

  it("should not be visible to a viewer", async () => {
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Viewer"),
    });
    const { result } = renderVisible(findSelectable(), table.TYPE_ONTOLOGY_ID, wrapper);
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
