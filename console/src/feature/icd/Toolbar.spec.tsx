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

import { ICD } from "@/feature/icd";
import { client, createICD } from "@/feature/icd/testutil";
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
  const proj = await client.projects.create({ name: uniqueName("project") });
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
  });
  render(
    <>
      {ICD.TOOLBAR.content}
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

describe("ICD toolbar", () => {
  it("should open an ICD in a tab on double click", async () => {
    const ic = await createICD();
    const { store } = await renderToolbar();
    fireEvent.doubleClick(await screen.findByRole("option", { name: ic.name }));
    expect(await focusedResourceKey(store)).toBe(ic.key);
  });

  describe("context menu", () => {
    it("should open an ICD from Open", async () => {
      const ic = await createICD();
      const { store } = await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(ic.name));
      fireEvent.click(await screen.findByText("Open"));
      expect(await focusedResourceKey(store)).toBe(ic.key);
    });

    it("should rename an ICD through the inline editor", async () => {
      const ic = await createICD();
      await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(ic.name));
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(`icd-name-${ic.key}`);
      const name = uniqueName("renamed");
      commitTextEdit(editor, name);
      await waitFor(async () =>
        expect((await client.icds.retrieve({ key: ic.key })).name).toBe(name),
      );
    });

    it("should delete an ICD after confirmation", async () => {
      const ic = await createICD();
      await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(ic.name));
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${ic.name}?`);
      fireEvent.click(findLastButton("Delete"));
      await waitFor(
        async () =>
          await expect(client.icds.retrieve({ key: ic.key })).rejects.toSatisfy((e) =>
            NotFoundError.matches(e),
          ),
      );
    });
  });
});
