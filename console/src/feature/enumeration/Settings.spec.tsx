// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Enumeration } from "@/feature/enumeration";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import {
  awaitGranted,
  awaitTextEditingElement,
  CaptureStatuses,
  clickAndSettle,
  commitTextEdit,
  createConsoleWrapper,
  HaulProvider,
  openContextMenu,
} from "@/testutil";

const client = createTestClient();
// Reads what the Core broadcasts, without the edits that the cache of client shows at
// once. Its cache hears a change a moment after the write that made it returns.
const core = createTestClient();

const createEnumeration = async (
  entries: enumeration.Entry[] = [
    { key: "a", value: 0, name: uniqueName("a"), abnormal: false },
    { key: "b", value: 1, name: uniqueName("b"), abnormal: false },
    { key: "c", value: 2, name: uniqueName("c"), abnormal: false },
  ],
): Promise<enumeration.Enumeration> =>
  await client.enumerations.create({ name: uniqueName("enumeration"), entries });

/** Creates a schematic whose one select draws the enumeration. */
const createSelectUser = async (e: enumeration.Key): Promise<schematic.Schematic> => {
  const project = await client.projects.create({ name: uniqueName("project") });
  return await client.schematics.create({
    project: project.key,
    name: uniqueName("schematic"),
    nodes: [
      {
        key: "select",
        position: { x: 0, y: 0 },
        zIndex: 0,
        config: schematic.nodeConfigZ.parse({ variant: "select", enumeration: e }),
      },
    ],
  });
};

/** Returns the nth built-in from the end, which no other spec edits. */
const builtinFromEnd = async (n: number): Promise<enumeration.Enumeration> => {
  const shipped = (await client.enumerations.retrieveBuiltins()).at(-n);
  if (shipped == null) throw new Error("the Core ships no enumerations");
  return shipped;
};

const renderEnumerations = async (onStatuses = vi.fn()): Promise<void> => {
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Tree.Provider items={{}}>
        <HaulProvider>{children}</HaulProvider>
        <Modals.Stack />
      </Tree.Provider>
      <CaptureStatuses onStatuses={onStatuses} />
    </Console>
  );
  await act(async () => {
    render(<Enumeration.Settings />, { wrapper });
  });
};

/** Returns the section that holds an enumeration's heading and entries. */
const findSection = async (name: string): Promise<HTMLElement> =>
  await screen.findByRole("region", { name });

/** Opens the menu of a built-in, which a user enumeration can share a name with. */
const openBuiltinMenu = async (name: string): Promise<void> => {
  const sections = await screen.findAllByRole("region", { name });
  const section = sections.find((r) => within(r).queryByText("Built-in") != null);
  if (section == null) throw new Error(`no built-in named ${name}`);
  await openContextMenu(() => within(section).getByText(name));
};

const stored = async (key: enumeration.Key): Promise<enumeration.Entry[]> =>
  (await core.enumerations.retrieve(key)).entries;

type Queries = ReturnType<typeof within>;

/**
 * Returns the queries of the dialog that holds the element. They stay fast while the
 * page shows each enumeration on the Core.
 */
const dialogOf = (el: HTMLElement): Queries => {
  const dialog = screen.getAllByRole("dialog").find((d) => d.contains(el));
  if (dialog == null) throw new Error("expected a dialog");
  return within(dialog);
};

const openDelete = async (name: string): Promise<Queries> => {
  await openContextMenu(name);
  await clickAndSettle("Delete");
  return dialogOf(await screen.findByText(/Are you sure you want to delete/));
};

const pickReplacement = async (dialog: Queries, name: string): Promise<void> => {
  fireEvent.click(await dialog.findByRole("button", { name: /Replace with/ }));
  const search = await screen.findByPlaceholderText("Search enumerations...");
  fireEvent.change(search, { target: { value: name } });
  const dropdown = dialogOf(search);
  fireEvent.click(await dropdown.findByRole("option", { name: new RegExp(name) }));
};

// The delete dialog enables its button once it knows what uses the enumeration.
const clickWhenEnabled = async (dialog: Queries, name: string): Promise<void> => {
  const button = await dialog.findByRole("button", { name });
  await waitFor(() => expect(button.getAttribute("aria-disabled")).toBeNull());
  await act(async () => {
    fireEvent.click(button);
  });
};

/** Waits for the delete dialog to close, which it does once the Core commits. */
const waitForDialogToClose = async (): Promise<void> =>
  await waitFor(() =>
    expect(screen.queryByText(/Are you sure you want to delete/)).toBeNull(),
  );

describe("Enumeration.Settings", () => {
  it("should create an enumeration and count up its entry values", async () => {
    await renderEnumerations();
    await clickAndSettle("Create enumeration");
    const name = uniqueName("enumeration");
    const field = await screen.findByLabelText("Enumeration name");
    fireEvent.change(field, { target: { value: name } });
    await act(async () => {
      fireEvent.keyDown(field, { key: "Enter" });
    });
    const section = await findSection(name);
    for (let i = 0; i < 2; i++)
      await act(async () => {
        fireEvent.click(within(section).getByRole("button", { name: "Add entry" }));
      });
    const created = (await client.enumerations.retrieve({ searchTerm: name })).find(
      (x) => x.name === name,
    );
    if (created == null) throw new Error(`the Core holds no ${name}`);
    await expect
      .poll(async () => (await stored(created.key)).map((e) => e.value))
      .toEqual([0, 1]);
  });

  it("should save an entry edited under its enumeration's heading", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const input = await within(await findSection(e.name)).findByLabelText("Name 2");
    const name = uniqueName("renamed");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: name } });
    fireEvent.blur(input);
    await expect
      .poll(async () => (await stored(e.key)).map((en) => en.name))
      .toEqual([e.entries[0].name, name, e.entries[2].name]);
  });

  it("should hide an enumeration's entries when its heading closes", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const section = await findSection(e.name);
    expect(await within(section).findByLabelText("Name 1")).toBeTruthy();
    fireEvent.click(
      within(section).getByRole("button", { name: new RegExp(e.name), expanded: true }),
    );
    expect(within(section).queryByLabelText("Name 1")).toBeNull();
    expect(section.classList).toContain(
      "console-enumeration-settings__section--closed",
    );
  });

  it("should rename an enumeration from its menu", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    await openContextMenu(e.name);
    await clickAndSettle("Rename");
    const name = uniqueName("renamed");
    commitTextEdit(await awaitTextEditingElement(), name);
    await expect
      .poll(async () => (await client.enumerations.retrieve(e.key)).name)
      .toBe(name);
  });

  it("should mark an edited built-in and restore its shipped entries", async () => {
    const shipped = await builtinFromEnd(2);
    await client.enumerations.retrieve(shipped.key);
    await client.enumerations.dispatch(
      shipped.key,
      enumeration.set({ path: ["entries"], value: shipped.entries.slice(1) }),
    );
    await renderEnumerations();
    expect(await screen.findByText("Edited")).toBeTruthy();
    await openBuiltinMenu(shipped.name);
    await clickAndSettle("Restore default");
    await expect.poll(async () => await stored(shipped.key)).toEqual(shipped.entries);
    await waitFor(() => expect(screen.queryByText("Edited")).toBeNull());
  });

  it("should not offer to delete a built-in", async () => {
    const shipped = await builtinFromEnd(3);
    await awaitGranted(client, enumeration.resourceID(shipped.key), "delete");
    await renderEnumerations();
    await openBuiltinMenu(shipped.name);
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should move a select to the enumeration picked on delete", async () => {
    const [from, to] = await Promise.all([createEnumeration(), createEnumeration()]);
    const user = await createSelectUser(from.key);
    await renderEnumerations();
    const dialog = await openDelete(from.name);
    expect(
      await within(await dialog.findByLabelText("Users")).findByText(user.name),
    ).toBeTruthy();
    await pickReplacement(dialog, to.name);
    await clickWhenEnabled(dialog, "Replace");
    await waitForDialogToClose();
    await expect
      .poll(async () => (await core.schematics.retrieve(user.key)).nodes[0].config)
      .toMatchObject({ variant: "select", enumeration: to.key });
  });

  it("should show a missing replacement at its field and keep the dialog open", async () => {
    const [from, to] = await Promise.all([createEnumeration(), createEnumeration()]);
    await createSelectUser(from.key);
    await renderEnumerations();
    const dialog = await openDelete(from.name);
    expect(await dialog.findByLabelText("Users")).toBeTruthy();
    await clickWhenEnabled(dialog, "Replace");
    expect(await dialog.findByText("Replacement is required")).toBeTruthy();
    expect(screen.getByText(/Are you sure you want to delete/)).toBeTruthy();
    await pickReplacement(dialog, to.name);
    expect(dialog.queryByText("Replacement is required")).toBeNull();
    expect((await core.enumerations.retrieve(from.key)).key).toEqual(from.key);
  });

  it("should open delete as a narrow confirmation with selectable text", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    await openContextMenu(e.name);
    await clickAndSettle("Delete");
    const message = await screen.findByText(/Are you sure you want to delete/);
    const dialog = (await screen.findAllByRole("dialog")).find((d) =>
      d.contains(message),
    );
    expect(dialog?.classList).toContain("console-confirm");
    expect(message.parentElement?.classList).toContain("console-confirm__body");
  });

  it("should delete an unused enumeration with no replace field", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    await openContextMenu(e.name);
    const item = await screen.findByText("Delete");
    // A synchronous act renders the dialog before the Core can say what uses it.
    act(() => {
      fireEvent.click(item);
    });
    const dialog = dialogOf(screen.getByText(/This action cannot be undone/));
    expect(dialog.queryByText(/Nothing uses it/)).toBeNull();
    expect(await dialog.findByText(/Nothing uses it/)).toBeTruthy();
    expect(dialog.queryByRole("button", { name: /Replace with/ })).toBeNull();
    await clickWhenEnabled(dialog, "Delete");
    await waitForDialogToClose();
    await expect(client.enumerations.retrieve(e.key)).rejects.toThrow(
      "enumeration was deleted",
    );
  });
});
