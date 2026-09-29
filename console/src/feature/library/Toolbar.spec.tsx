// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Library } from "@/feature/library";
import { client, createLibrary } from "@/feature/library/testutil";
import { Modals } from "@/platform/modals";
import { findLastButton } from "@/platform/modals/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import {
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  resolveFocusedTab,
  type TestStore,
  uniqueName,
} from "@/testutil";

const renderToolbar = async (): Promise<{ store: TestStore }> => {
  const proj = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
  });
  render(
    <>
      {Library.TOOLBAR.content}
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store };
};

const focusedResourceKey = async (store: TestStore): Promise<string> => {
  const tab = await resolveFocusedTab(store, client);
  if (tab.variant !== "resource") throw new Error("expected a resource tab");
  return tab.resource.key;
};

describe("Library toolbar", () => {
  it("should open a library in a tab on double click", async () => {
    const lib = await createLibrary();
    const { store } = await renderToolbar();
    fireEvent.doubleClick(await screen.findByText(lib.name));
    expect(await focusedResourceKey(store)).toBe(lib.key);
  });

  describe("context menu", () => {
    it("should open a library from Open", async () => {
      const lib = await createLibrary();
      const { store } = await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(lib.name));
      fireEvent.click(await screen.findByText("Open"));
      expect(await focusedResourceKey(store)).toBe(lib.key);
    });

    it("should rename a library through the inline editor", async () => {
      const lib = await createLibrary();
      await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(lib.name));
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(`library-name-${lib.key}`);
      const name = uniqueName("renamed");
      commitTextEdit(editor, name);
      await waitFor(async () =>
        expect((await client.libraries.retrieve({ key: lib.key })).name).toBe(name),
      );
    });

    it("should delete a library after confirmation", async () => {
      const lib = await createLibrary();
      await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(lib.name));
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${lib.name}?`);
      fireEvent.click(findLastButton("Delete"));
      await waitFor(
        async () =>
          await expect(client.libraries.retrieve({ key: lib.key })).rejects.toSatisfy(
            (e) => NotFoundError.matches(e),
          ),
      );
    });
  });
});
