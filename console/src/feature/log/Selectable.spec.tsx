// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import {
  assertDefined,
  createConsoleWrapper,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("log/Selectable", () => {
  it("creates a log in the active project and opens its tab when clicked", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    const Selectable = Log.SELECTABLES[0];
    expect(Selectable.type).toBe(log.TYPE_ONTOLOGY_ID.type);
    render(<Selectable />, { wrapper });
    fireEvent.click(await screen.findByText("Log"));
    const tab = await resolveFocusedTab(store, client, (t) => t.variant === "resource");
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(log.TYPE_ONTOLOGY_ID.type);
    const key = tab.resource.key;
    const created = await client.logs.retrieve(key);
    expect(created.name).toBe("Log");
    await waitFor(() =>
      expect(
        documentIn(Session.Log.selectSliceState(store.getState()), key),
      ).toBeDefined(),
    );
  });
});

describe("log/Selectable permissions", () => {
  const findSelectable = () => {
    const Selectable = Log.SELECTABLES.find(
      (s) => s.type === log.TYPE_ONTOLOGY_ID.type,
    );
    assertDefined(Selectable, "no selectable registered for log");
    return Selectable;
  };

  it("should offer the tile to an engineer", async () => {
    const Selectable = findSelectable();
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Engineer"),
    });
    render(<Selectable />, { wrapper });
    expect(await screen.findByText("Log")).toBeTruthy();
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
