// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { client, testProjectKey } from "@/feature/schematic/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { assertDefined, createConsoleWrapper, resolveFocusedTab } from "@/testutil";

const roles = new RoleClients(client);

describe("schematic/Selectable", () => {
  it("creates a schematic in the active project and opens its tab when clicked", async () => {
    const proj = await client.projects.retrieve(await testProjectKey());
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    const Selectable = Schematic.SELECTABLES[0];
    expect(Selectable.type).toBe(schematic.TYPE_ONTOLOGY_ID.type);
    render(<Selectable />, { wrapper });
    fireEvent.click(await screen.findByText("Schematic"));
    const tab = await resolveFocusedTab(store, client, (t) => t.variant === "resource");
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(schematic.TYPE_ONTOLOGY_ID.type);
    const key = tab.resource.key;
    const created = await client.schematics.retrieve(key);
    expect(created.name).toBe("Schematic");
    await waitFor(() =>
      expect(Session.Schematic.selectEditable({ state: store.getState(), key })).toBe(
        true,
      ),
    );
  });
});

describe("schematic/Selectable permissions", () => {
  const findSelectable = () => {
    const Selectable = Schematic.SELECTABLES.find(
      (s) => s.type === schematic.TYPE_ONTOLOGY_ID.type,
    );
    assertDefined(Selectable, "no selectable registered for schematic");
    return Selectable;
  };

  it("should offer the tile to an engineer", async () => {
    const Selectable = findSelectable();
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Engineer"),
    });
    render(<Selectable />, { wrapper });
    expect(await screen.findByText("Schematic")).toBeTruthy();
  });

  it("should withhold the tile from a viewer", async () => {
    const Selectable = findSelectable();
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Viewer"),
    });
    const { container } = render(<Selectable />, { wrapper });
    await waitFor(() => expect(container.textContent).toBe(""));
  });
});
