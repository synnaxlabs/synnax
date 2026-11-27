// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, NotFoundError, panel, project } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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
  createTestClientWithGrants,
  resolveFocusedTab,
  type TestStore,
} from "@/testutil";

const TAB_TYPES = [project.TYPE_RESOURCE_ID, panel.TYPE_RESOURCE_ID];

// The page lists every ICD its user can read, and other specs leave theirs on the
// Core. A user that reads only the ICD of the test sees the same rows on any Core.
const renderSettings = async (shown: icd.ICD): Promise<{ store: TestStore }> => {
  const proj = await client.projects.create({ name: uniqueName("project") });
  const subject = await createTestClientWithGrants(client, {
    retrieve: [icd.resourceID(shown.key), ...TAB_TYPES],
    create: TAB_TYPES,
    update: [icd.TYPE_RESOURCE_ID, ...TAB_TYPES],
    delete: [icd.TYPE_RESOURCE_ID],
  });
  const { wrapper, store } = await createConsoleWrapper({
    client: subject,
    preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
  });
  render(
    <>
      <ICD.Settings />
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

describe("ICD.Settings", () => {
  it("should list an ICD with its message count", async () => {
    const ic = await createICD();
    await renderSettings(ic);
    const item = await screen.findByRole("option", { name: ic.name });
    expect(within(item).getByText("0")).toBeTruthy();
  });

  it("should open an ICD in a tab on click", async () => {
    const ic = await createICD();
    const { store } = await renderSettings(ic);
    fireEvent.click(await screen.findByRole("option", { name: ic.name }));
    expect(await focusedResourceKey(store)).toBe(ic.key);
  });

  describe("context menu", () => {
    it("should open an ICD from Open", async () => {
      const ic = await createICD();
      const { store } = await renderSettings(ic);
      fireEvent.contextMenu(await screen.findByText(ic.name));
      fireEvent.click(await screen.findByText("Open"));
      expect(await focusedResourceKey(store)).toBe(ic.key);
    });

    it("should rename an ICD through the inline editor", async () => {
      const ic = await createICD();
      await renderSettings(ic);
      fireEvent.contextMenu(await screen.findByText(ic.name));
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(List.itemNameID(ic.key));
      const name = uniqueName("renamed");
      commitTextEdit(editor, name);
      await waitFor(async () =>
        expect((await client.icds.retrieve({ key: ic.key })).name).toBe(name),
      );
    });

    it("should delete an ICD after confirmation", async () => {
      const ic = await createICD();
      await renderSettings(ic);
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
