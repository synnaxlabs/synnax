// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Selector } from "@/feature/panel/Selector";
import { createPanelWrapper } from "@/platform/panel/testutil";
import { Session } from "@/session";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Panel.Selector", () => {
  it("should select a newly created panel", async () => {
    const { wrapper, store } = await createPanelWrapper({ client });
    await act(async () => {
      render(<Selector />, { wrapper });
    });
    // The strip suspends on the project's panel list, so nothing renders until it
    // resolves.
    const create = await screen.findByRole("button");
    expect(Session.Panel.selectSelected(store.getState())).toBeUndefined();
    await act(async () => {
      fireEvent.click(create);
    });
    const selected = Session.Panel.selectSelected(store.getState());
    expect(selected).toBeDefined();
    await waitFor(() => expect(screen.getByText("New Panel")).toBeTruthy());
  });

  // The membership query answers in an order the user never chose; the session's
  // strip order is what the pills must follow.
  it("should render the pills in the session's strip order", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const createPanel = async (name: string): Promise<panel.Panel> =>
      await client.panels.create({
        name,
        parent: project.ontologyID(proj.key),
        root: { variant: "leaf", tabs: [] },
      });
    const alpha = await createPanel(uniqueName("alpha"));
    const bravo = await createPanel(uniqueName("bravo"));
    const { wrapper, store } = await createPanelWrapper({
      client,
      project: proj.key,
    });
    act(() => {
      store.dispatch(
        Session.Panel.reconcileOrder({
          panels: [
            { key: alpha.key, name: alpha.name },
            { key: bravo.key, name: bravo.name },
          ],
        }),
      );
      store.dispatch(Session.Panel.reorder({ key: bravo.key, index: 0 }));
    });
    await act(async () => {
      render(<Selector />, { wrapper });
    });
    await screen.findByText(alpha.name);
    await screen.findByText(bravo.name);
    const tabs = screen.getAllByRole("tab");
    expect(tabs.map((tab) => tab.textContent)).toEqual([bravo.name, alpha.name]);
  });
});
