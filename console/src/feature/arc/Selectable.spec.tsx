// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Modals } from "@/platform/modals";
import { assertDefined, createConsoleWrapper } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("arc/Selectable", () => {
  it("opens the arc create modal when the tile is clicked", async () => {
    const Selectable = Arc.SELECTABLES.find(
      (s) => s.type === arc.TYPE_ONTOLOGY_ID.type,
    );
    assertDefined(Selectable, "no selectable registered for the arc editor");
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <>
        <Selectable />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    fireEvent.click(await screen.findByText("Arc automation"));
    expect(await screen.findByPlaceholderText("Name")).toBeTruthy();
  });
});

describe("arc/Selectable permissions", () => {
  const findSelectable = () => {
    const Selectable = Arc.SELECTABLES.find(
      (s) => s.type === arc.TYPE_ONTOLOGY_ID.type,
    );
    assertDefined(Selectable, "no selectable registered for arc");
    return Selectable;
  };

  it("should offer the tile to an engineer", async () => {
    const Selectable = findSelectable();
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Engineer"),
    });
    render(<Selectable />, { wrapper });
    expect(await screen.findByText("Arc automation")).toBeTruthy();
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
