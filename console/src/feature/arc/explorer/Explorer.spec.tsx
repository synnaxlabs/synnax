// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import {
  awaitTextEditing,
  commitTextEdit,
  countEditableText,
  createConsoleWrapper,
  createTestClientWithGrants,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createArc = async () =>
  await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [] },
  });

const renderExplorer = async (as: Synnax = client) => {
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  render(<Arc.Explorer.Explorer />, { wrapper });
  return { store };
};

describe("arc/Explorer", () => {
  it("surfaces Arcs matching the search term and hides the rest", async () => {
    const a = await createArc();
    const b = await createArc();
    await renderExplorer();
    expect(await screen.findByText("Search Arcs...")).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox"), { target: { value: a.name } });
    expect(await screen.findByText(a.name)).toBeTruthy();
    expect(screen.queryByText(b.name)).toBeNull();
  });

  it("renames an Arc through the context menu inline editor", async () => {
    const arc = await createArc();
    await renderExplorer();
    fireEvent.change(screen.getByRole("textbox"), { target: { value: arc.name } });
    fireEvent.contextMenu(await screen.findByText(arc.name));
    fireEvent.click(await screen.findByText("Rename"));
    const editor = await awaitTextEditing(`arc-explorer-text-${arc.key}`);
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.arcs.retrieve(arc.key)).name).toBe(renamed),
    );
  });

  it("renders the Arc name as plain text for a subject who cannot update Arcs", async () => {
    const arc = await createArc();
    await renderExplorer(
      await createTestClientWithGrants(client, {
        retrieve: [clientArc.TYPE_ONTOLOGY_ID],
      }),
    );
    fireEvent.change(screen.getByRole("textbox"), { target: { value: arc.name } });
    await screen.findByText(arc.name);
    await act(async () => {});
    expect(countEditableText(`arc-explorer-text-${arc.key}`)).toBe(0);
  });
});
