// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, view } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { act, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { assert, describe, expect, it, onTestFinished } from "vitest";

import { View } from "@/platform/view";
import { createTabWrapper } from "@/platform/view/testutil";
import { renderSuspended, uniqueName } from "@/testutil";

const client = createTestClient();

const TAB = View.createTab({
  range: {
    Content: () => <>range explorer</>,
    Name: () => <>Range explorer</>,
    Icon: Icon.Range,
  },
  status: {
    Content: () => <>status explorer</>,
    Name: () => <>Status explorer</>,
    Icon: Icon.Status,
  },
});

const createView = async (type: view.Type = "range"): Promise<view.View> => {
  const created = await client.views.create({ name: uniqueName("view"), type });
  onTestFinished(async () => await client.views.delete(created.key));
  return created;
};

/** Renders the element in a tab that shows the saved view. */
const renderShowing = async (saved: view.View, ui: ReactElement): Promise<void> => {
  const { wrapper } = await createTabWrapper({
    client,
    tab: { variant: "resource", resource: view.resourceID(saved.key) },
  });
  await renderSuspended(ui, { wrapper });
};

describe("View.createTab", () => {
  it("shows the explorer of the type of its view", async () => {
    await renderShowing(await createView("status"), <TAB.Content />);
    expect(await screen.findByText("status explorer")).toBeTruthy();
  });

  it("names the tab after the type and the name of its view", async () => {
    const saved = await createView("status");
    await renderShowing(saved, <TAB.Name />);
    expect(await screen.findByText(`Statuses › ${saved.name}`)).toBeTruthy();
  });

  it("follows a rename of its view", async () => {
    const saved = await createView();
    await renderShowing(saved, <TAB.Name />);
    await screen.findByText(`Ranges › ${saved.name}`);
    const renamed = uniqueName("renamed");
    await act(async () => await client.views.rename(saved.key, renamed));
    expect(await screen.findByText(`Ranges › ${renamed}`)).toBeTruthy();
  });

  it("restores a deleted view under its original key", async () => {
    const { restore } = TAB;
    assert(restore != null);
    const saved = await createView();
    // Restore rebuilds from the corpse the cache holds, which exists only once the
    // client has seen the view live.
    await client.views.retrieve({ key: saved.key });
    await client.views.delete(saved.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.views.watch(saved.key).getCached())).toBe(
        true,
      ),
    );
    const { key: project } = await client.projects.create({ name: uniqueName("proj") });
    await restore({ client, project, resource: view.resourceID(saved.key) });
    // The restoring client writes through its own cache, so a second client reads it.
    const restored = await createTestClient().views.retrieve({ key: saved.key });
    expect(restored.name).toEqual(saved.name);
  });
});
