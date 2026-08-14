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
import { Haul } from "@synnaxlabs/pluto";
import { fireDragEvent } from "@synnaxlabs/pluto/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Selector } from "@/feature/panel/Selector";
import { createPanelWrapper } from "@/platform/panel/testutil";
import { Session } from "@/session";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const createProjectPanel = async (
  projectKey: project.Key,
  name: string,
  key: panel.Key = uuid.create(),
): Promise<panel.Panel> =>
  await client.panels.create({
    key,
    name: uniqueName(name),
    parent: project.ontologyID(projectKey),
    root: { variant: "leaf", tabs: [] },
  });

/**
 * Nests the wrapper in the haul provider the app mounts through Pluto.Context, so a
 * pill drag resolves against the same redux-backed drag state as production.
 */
const withHaul = (Wrapper: FC<PropsWithChildren>): FC<PropsWithChildren> => {
  const HaulWrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Wrapper>
      <Haul.Provider {...Session.Haul.PROVIDER_PROPS}>{children}</Haul.Provider>
    </Wrapper>
  );
  HaulWrapper.displayName = "HaulWrapper";
  return HaulWrapper;
};

const pillNames = (): (string | null)[] =>
  screen.getAllByRole("tab").map((tab) => tab.textContent);

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
    const alpha = await createProjectPanel(proj.key, "alpha");
    const bravo = await createProjectPanel(proj.key, "bravo");
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
    expect(pillNames()).toEqual([bravo.name, alpha.name]);
  });

  // A panel the session has not reconciled yet must not displace the panels the
  // user ordered, however early it lands in the membership answer.
  it("should render a panel missing from the strip order last", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    // The membership answer arrives in key order, so the unreconciled panel takes
    // the lower key: a correct sort cannot pass by luck.
    const [lowKey, highKey] = [uuid.create(), uuid.create()].sort();
    const pending = await createProjectPanel(proj.key, "pending", lowKey);
    const known = await createProjectPanel(proj.key, "known", highKey);
    const { wrapper, store } = await createPanelWrapper({
      client,
      project: proj.key,
    });
    act(() => {
      store.dispatch(
        Session.Panel.reconcileOrder({
          panels: [{ key: known.key, name: known.name }],
        }),
      );
    });
    await act(async () => {
      render(<Selector />, { wrapper });
    });
    await screen.findByText(pending.name);
    await screen.findByText(known.name);
    expect(pillNames()).toEqual([known.name, pending.name]);
  });

  // Pluto resolves the insertion slot and the slice applies the move; nothing else
  // proves the strip hands one to the other, so a mis-wired drop is invisible.
  it("should reorder the strip when a pill is dragged past the last slot", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const alpha = await createProjectPanel(proj.key, "alpha");
    const bravo = await createProjectPanel(proj.key, "bravo");
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
    });
    await act(async () => {
      render(<Selector />, { wrapper: withHaul(wrapper) });
    });
    await screen.findByText(alpha.name);
    await screen.findByText(bravo.name);
    expect(pillNames()).toEqual([alpha.name, bravo.name]);
    const [first] = screen.getAllByRole("tab");
    // Every element shares one faked 100-wide rect, so a cursor past the common
    // center of the pills resolves to the slot after the last one.
    act(() => {
      fireEvent.dragStart(first);
      fireDragEvent(screen.getByRole("tablist"), "drop", { x: 200, y: 16 });
    });
    await waitFor(() => expect(pillNames()).toEqual([bravo.name, alpha.name]));
    expect(Session.Panel.selectOrder(store.getState())).toEqual([bravo.key, alpha.key]);
  });
});
