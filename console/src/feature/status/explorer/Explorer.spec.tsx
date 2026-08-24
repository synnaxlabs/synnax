// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  group,
  label,
  type ontology,
  status,
  type Synnax as Client,
  view,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Status } from "@/feature/status";
import { Modals } from "@/platform/modals";
import { enableEditing, findToolbarIconButton } from "@/platform/view/testutil";
import {
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  createTestClientWithGrants,
  getBySelector,
  queryIconButton,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

describe("status explorer", () => {
  it("should list statuses matching a search", async () => {
    const s = await client.statuses.set(
      status.create({ name: uniqueName("status"), variant: "info", message: "m" }),
    );
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <>
        <Status.Explorer.Explorer />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    await enableEditing();
    const search = await screen.findByPlaceholderText("Search statuses...");
    fireEvent.change(search, { target: { value: s.name } });
    expect(await screen.findByText(s.name)).toBeTruthy();
  });

  it("should rename a status in place from the context menu", async () => {
    const s = await client.statuses.set(
      status.create({ name: uniqueName("status"), variant: "info", message: "m" }),
    );
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <>
        <Status.Explorer.Explorer />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    await enableEditing();
    const search = await screen.findByPlaceholderText("Search statuses...");
    fireEvent.change(search, { target: { value: s.name } });
    fireEvent.contextMenu(await screen.findByText(s.name));
    fireEvent.click(await screen.findByText("Rename"));
    const editor = await awaitTextEditing(List.itemNameID(s.key));
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.statuses.retrieve(s.key)).name).toBe(renamed),
    );
  });
});

describe("status explorer permissions", () => {
  const createEditor = async (creatable: ontology.ID[] = []) =>
    await createTestClientWithGrants(client, {
      retrieve: [
        status.TYPE_ONTOLOGY_ID,
        view.TYPE_ONTOLOGY_ID,
        label.TYPE_ONTOLOGY_ID,
        group.TYPE_ONTOLOGY_ID,
      ],
      update: [view.TYPE_ONTOLOGY_ID],
      create: [view.TYPE_ONTOLOGY_ID, ...creatable],
    });

  const renderExplorer = async (as: Client) => {
    const { wrapper } = await createConsoleWrapper({ client: as });
    render(
      <>
        <Status.Explorer.Explorer />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    await enableEditing();
  };

  it("should withhold the create button from a subject who cannot create statuses", async () => {
    await renderExplorer(await createEditor());
    await waitFor(() =>
      expect(
        queryIconButton(getBySelector(document.body, ".console-controls"), "add"),
      ).toBeTruthy(),
    );
    expect(
      queryIconButton(getBySelector(document.body, ".console-view__toolbar"), "add"),
    ).toBeNull();
  });

  it("should offer the create button to a subject who may create statuses", async () => {
    await renderExplorer(await createEditor([status.TYPE_ONTOLOGY_ID]));
    expect(await findToolbarIconButton("add")).toBeTruthy();
  });
});
