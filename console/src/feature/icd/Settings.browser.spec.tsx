// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  icd,
  NotFoundError,
  panel,
  project,
  type resource,
  type Synnax,
  task,
} from "@synnaxlabs/client";
import { failRequests, holdRequests } from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/lyra/list";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ICD } from "@/feature/icd";
import { client, createICD, createUsedICD } from "@/feature/icd/testutil";
import { Task } from "@/feature/task";
import { createdIn } from "@/platform/icd/testutil";
import { Modals } from "@/platform/modals";
import { findLastButton } from "@/platform/modals/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Tree } from "@/platform/tree";
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

interface RenderSettingsOptions {
  /** Other resources that the user reads. */
  retrieve?: resource.ID[];
  canCreate?: boolean;
}

// The page lists every ICD its user can read, and other specs leave theirs on the
// Core. A user that reads only the ICDs of the test sees the same rows on any Core. A
// user that creates ICDs reads them all.
const renderSettings = async (
  shown: icd.ICD | icd.ICD[],
  { retrieve = [], canCreate = false }: RenderSettingsOptions = {},
): Promise<{ store: TestStore; subject: Synnax }> => {
  const proj = await client.projects.create({ name: uniqueName("project") });
  const keys = (Array.isArray(shown) ? shown : [shown]).map(({ key }) => key);
  const subject = await createTestClientWithGrants(client, {
    retrieve: [
      ...(canCreate ? [icd.TYPE_RESOURCE_ID] : icd.resourceID(keys)),
      ...retrieve,
      ...TAB_TYPES,
    ],
    create: canCreate ? [icd.TYPE_RESOURCE_ID, ...TAB_TYPES] : TAB_TYPES,
    update: [icd.TYPE_RESOURCE_ID, ...TAB_TYPES],
    delete: [icd.TYPE_RESOURCE_ID],
  });
  const { wrapper, store } = await createConsoleWrapper({
    client: subject,
    preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
  });
  // The delete dialog lists the tasks that use an ICD as tree items.
  render(
    <Tree.Provider items={Task.TREE_ITEMS}>
      <ICD.Settings />
      <Modals.Stack />
    </Tree.Provider>,
    { wrapper },
  );
  return { store, subject };
};

const openDelete = async (name: string): Promise<void> => {
  fireEvent.contextMenu(await screen.findByText(name));
  fireEvent.click(await screen.findByText("Delete"));
};

const deleteDisabled = (): boolean =>
  findLastButton("Delete").getAttribute("aria-disabled") === "true";

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

  it("should open a created ICD in a tab, marked to focus its name", async () => {
    const ic = await createICD();
    const { store } = await renderSettings(ic, { canCreate: true });
    fireEvent.click(await screen.findByRole("button", { name: "Create ICD" }));
    const key = await focusedResourceKey(store);
    expect(key).not.toBe(ic.key);
    expect(createdIn(store, key)).toBe(true);
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
      await openDelete(ic.name);
      await screen.findByText(`Are you sure you want to delete ${ic.name}?`);
      await waitFor(() => expect(deleteDisabled()).toBe(false));
      fireEvent.click(findLastButton("Delete"));
      await waitFor(
        async () =>
          await expect(client.icds.retrieve({ key: ic.key })).rejects.toSatisfy((e) =>
            NotFoundError.matches(e),
          ),
      );
    });

    it("should name the tasks that use an ICD and offer no delete", async () => {
      const { ic, used } = await createUsedICD();
      await renderSettings(ic, { retrieve: [task.resourceID(used.key)] });
      await openDelete(ic.name);
      await screen.findByText(`${ic.name} is in use`);
      expect(screen.getByText("Remove it from these tasks first.")).toBeTruthy();
      expect(await screen.findByText(used.name)).toBeTruthy();
      expect(deleteDisabled()).toBe(true);
    });

    it("should say that some of several ICDs are in use", async () => {
      const { ic, used } = await createUsedICD();
      const unused = await createICD();
      await renderSettings([ic, unused], {
        retrieve: [task.resourceID(used.key)],
      });
      for (const { name } of [ic, unused]) {
        const option = await screen.findByRole("option", { name });
        fireEvent.click(within(option).getByRole("checkbox", { name: "Select" }));
      }
      fireEvent.contextMenu(await screen.findByText(unused.name));
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText("Some of these ICDs are in use");
      expect(screen.getByText("Remove them from these tasks first.")).toBeTruthy();
      expect(deleteDisabled()).toBe(true);
    });

    it("should open a task from the delete dialog and close the dialog", async () => {
      const { ic, used } = await createUsedICD();
      const { store } = await renderSettings(ic, {
        retrieve: [task.resourceID(used.key)],
      });
      await openDelete(ic.name);
      fireEvent.click(await screen.findByText(used.name));
      expect(await focusedResourceKey(store)).toBe(task.resourceID(used.key).key);
      await waitFor(() =>
        expect(screen.queryByText(`${ic.name} is in use`)).toBeNull(),
      );
    });

    it("should say why it cannot find the tasks that use an ICD and offer no delete", async () => {
      const ic = await createICD();
      const { subject } = await renderSettings(ic);
      await screen.findByText(ic.name);
      failRequests(subject, "/task/retrieve", new Error("task lookup failed"));
      await openDelete(ic.name);
      expect(
        await screen.findByText("Failed to retrieve tasks: task lookup failed"),
      ).toBeTruthy();
      expect(deleteDisabled()).toBe(true);
    });

    it("should offer delete only once it knows that no task uses the ICD", async () => {
      const ic = await createICD();
      const { subject } = await renderSettings(ic);
      await screen.findByText(ic.name);
      const { release } = holdRequests(subject, "/task/retrieve");
      await openDelete(ic.name);
      await screen.findByText(`Are you sure you want to delete ${ic.name}?`);
      expect(deleteDisabled()).toBe(true);
      release();
      await waitFor(() => expect(deleteDisabled()).toBe(false));
    });

    it("should offer delete when the user cannot see the task that uses an ICD", async () => {
      const { ic } = await createUsedICD();
      await renderSettings(ic);
      await openDelete(ic.name);
      await screen.findByText(`Are you sure you want to delete ${ic.name}?`);
      await waitFor(() => expect(deleteDisabled()).toBe(false));
    });
  });
});
