// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, type project, type view } from "@synnaxlabs/client";
import { createPanelProject, createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

let projectKey: project.Key;

beforeAll(async () => {
  await client.connect();
  projectKey = await createPanelProject(client);
});

const createView = async (): Promise<view.View> =>
  await client.views.create({ name: uniqueName("view"), type: "range" });

const createPanel = async (...tabs: panel.TabKey[]): Promise<panel.Panel> =>
  await client.panels.create({
    ...panel.panelZ.parse({
      name: uniqueName("panel"),
      root: {
        variant: "leaf",
        tabs: tabs.map((key) => ({ variant: "view", key, type: "range_explorer" })),
      },
    }),
    project: projectKey,
  });

const mount = async (open: Record<panel.TabKey, view.Key>) =>
  await renderHookWithConsole(
    () => Session.Synchronizer.use(Session.View.SYNCHRONIZERS),
    {
      client,
      preloadedState: {
        [Session.Project.SLICE_NAME]: {
          ...Session.Project.ZERO_SLICE_STATE,
          selected: projectKey,
        },
        [Session.View.SLICE_NAME]: { ...Session.View.ZERO_SLICE_STATE, open },
      },
    },
  );

const openOf = (state: Session.State): Record<panel.TabKey, view.Key> =>
  state[Session.View.SLICE_NAME].open;

describe("View.SYNCHRONIZERS", () => {
  it("should send a tab back to its static view when its view is deleted", async () => {
    const created = await createView();
    const tab = uuid.create();
    await createPanel(tab);
    const { store } = await mount({ [tab]: created.key });
    expect(openOf(store.getState())).toEqual({ [tab]: created.key });
    await client.views.delete(created.key);
    await waitFor(() => expect(openOf(store.getState())).toEqual({}));
  });

  it("should drop views deleted while away, keeping live ones", async () => {
    const live = await createView();
    const [kept, dropped] = [uuid.create(), uuid.create()];
    await createPanel(kept, dropped);
    const { store } = await mount({ [kept]: live.key, [dropped]: uuid.create() });
    await waitFor(() => expect(openOf(store.getState())).toEqual({ [kept]: live.key }));
  });

  it("should drop tabs closed while away, keeping open ones", async () => {
    const shown = await createView();
    const open = uuid.create();
    await createPanel(open);
    const { store } = await mount({ [open]: shown.key, [uuid.create()]: shown.key });
    await waitFor(() =>
      expect(openOf(store.getState())).toEqual({ [open]: shown.key }),
    );
  });
});
