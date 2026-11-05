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
import { screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Modals } from "@/platform/modals";
import { renderVisible, select } from "@/platform/selector/testutil";
import { assertDefined, createConsoleWrapper } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("arc/Selectable", () => {
  it("opens the arc create modal when selected", async () => {
    const Selectable = Arc.SELECTABLES.find(
      (s) => s.type === arc.TYPE_ONTOLOGY_ID.type,
    );
    assertDefined(Selectable, "no selectable registered for the arc editor");
    const { wrapper: Base } = await createConsoleWrapper({ client });
    const wrapper = ({ children }: PropsWithChildren) => (
      <Base>
        {children}
        <Modals.Stack />
      </Base>
    );
    select(Selectable, wrapper);
    expect(await screen.findByPlaceholderText("Arc name")).toBeTruthy();
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

  it("should be visible to an engineer", async () => {
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Engineer"),
    });
    const { result } = renderVisible(findSelectable(), arc.TYPE_ONTOLOGY_ID, wrapper);
    await waitFor(() => expect(result.current.visible).toBe(true));
  });

  it("should not be visible to a viewer", async () => {
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Viewer"),
    });
    const { result } = renderVisible(findSelectable(), arc.TYPE_ONTOLOGY_ID, wrapper);
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
