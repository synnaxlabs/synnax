// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration, schematic, type Synnax } from "@synnaxlabs/client";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
} from "@synnaxlabs/client/testutil";
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
import { describe, expect, it } from "vitest";

import { Enumeration } from "@/feature/enumeration";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import {
  awaitGranted,
  awaitTextEditingElement,
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

const renderEnumerations = async (as: Synnax = client): Promise<void> => {
  const { wrapper: Console } = await createConsoleWrapper({ client: as });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Tree.Provider items={{}}>
        <HaulProvider>{children}</HaulProvider>
        <Modals.Stack />
      </Tree.Provider>
    </Console>
  );
  await act(async () => {
    render(<Enumeration.Settings />, { wrapper });
  });
};

/** Opens an enumeration and returns the entries that show under it. */
const openEntries = async (name: string): Promise<HTMLElement> => {
  fireEvent.click(await screen.findByRole("option", { name }));
  return await screen.findByRole("region", { name: `${name} entries` });
};

/** Opens the menu of a built-in, which a user enumeration can share a name with. */
const openBuiltinMenu = async (name: string): Promise<void> => {
  const items = await screen.findAllByRole("option", { name });
  const item = items.find((o) => within(o).queryByText("Built-in") != null);
  if (item == null) throw new Error(`no built-in named ${name}`);
  await openContextMenu(() => within(item).getByText(name));
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

/** Waits for the cache of core to hear that the Core deleted the enumeration. */
const waitForDeleted = async (key: enumeration.Key): Promise<void> =>
  await waitFor(
    async () =>
      await expect(core.enumerations.retrieve(key)).rejects.toThrow(
        `enumeration with key ${key} not found`,
      ),
  );

describe("Enumeration.Settings", () => {
  /** Creates an enumeration through the create form and returns its entries. */
  const createThroughForm = async (name: string): Promise<HTMLElement> => {
    await clickAndSettle("Create enumeration");
    const field = await screen.findByLabelText("Enumeration name");
    fireEvent.change(field, { target: { value: name } });
    await act(async () => {
      fireEvent.keyDown(field, { key: "Enter" });
    });
    return await screen.findByRole("region", { name: `${name} entries` });
  };

  it("should open a new enumeration with Add entry focused", async () => {
    await renderEnumerations();
    const name = uniqueName("enumeration");
    const entries = await createThroughForm(name);
    const add = within(entries).getByRole("button", { name: "Add entry" });
    await waitFor(() => expect(document.activeElement).toBe(add));
    expect(screen.getByRole("option", { name }).getAttribute("aria-expanded")).toBe(
      "true",
    );
    // jsdom gives a focused button no native Enter activation, so a click stands in.
    await act(async () => {
      fireEvent.click(add);
    });
    await waitFor(() =>
      expect(document.activeElement).toBe(within(entries).getByLabelText("Name 1")),
    );
  });

  it("should keep focus in the search when a new enumeration shows again", async () => {
    await renderEnumerations();
    const name = uniqueName("enumeration");
    const entries = await createThroughForm(name);
    const add = within(entries).getByRole("button", { name: "Add entry" });
    await waitFor(() => expect(document.activeElement).toBe(add));
    const search = screen.getByPlaceholderText("Search enumerations");
    act(() => search.focus());
    fireEvent.change(search, { target: { value: uniqueName("zzz") } });
    await waitFor(() =>
      expect(screen.queryByRole("region", { name: `${name} entries` })).toBeNull(),
    );
    fireEvent.change(search, { target: { value: name } });
    await screen.findByRole("region", { name: `${name} entries` });
    await act(async () => {});
    expect(document.activeElement).toBe(search);
  });

  it("should not focus a hidden new enumeration that a search shows", async () => {
    await renderEnumerations();
    const search = screen.getByPlaceholderText("Search enumerations");
    fireEvent.change(search, { target: { value: uniqueName("zzz") } });
    const name = uniqueName("enumeration");
    await clickAndSettle("Create enumeration");
    const field = await screen.findByLabelText("Enumeration name");
    fireEvent.change(field, { target: { value: name } });
    await act(async () => {
      fireEvent.keyDown(field, { key: "Enter" });
    });
    // The create form closes once the Core holds the new enumeration.
    await waitFor(() => expect(field.isConnected).toBe(false));
    act(() => search.focus());
    fireEvent.change(search, { target: { value: name } });
    await screen.findByRole("region", { name: `${name} entries` });
    await act(async () => {});
    expect(document.activeElement).toBe(search);
  });

  it("should focus the entries of each new enumeration", async () => {
    await renderEnumerations();
    const first = await createThroughForm(uniqueName("enumeration"));
    const firstAdd = within(first).getByRole("button", { name: "Add entry" });
    await waitFor(() => expect(document.activeElement).toBe(firstAdd));
    const second = await createThroughForm(uniqueName("enumeration"));
    const add = within(second).getByRole("button", { name: "Add entry" });
    await waitFor(() => expect(document.activeElement).toBe(add));
  });

  it("should count the schematics that use an enumeration as they change", async () => {
    const e = await createEnumeration();
    const user = await createSelectUser(e.key);
    await renderEnumerations();
    const item = await screen.findByRole("option", { name: e.name });
    expect(await within(item).findByText("Used by 1")).toBeTruthy();
    await createSelectUser(e.key);
    expect(await within(item).findByText("Used by 2")).toBeTruthy();
    await client.schematics.delete(user.key);
    expect(await within(item).findByText("Used by 1")).toBeTruthy();
  });

  it("should list the schematics that use an enumeration from its count", async () => {
    const e = await createEnumeration();
    const user = await createSelectUser(e.key);
    await renderEnumerations();
    const item = await screen.findByRole("option", { name: e.name });
    const count = await within(item).findByText("Used by 1");
    expect(screen.queryByText(user.name)).toBeNull();
    await act(async () => {
      fireEvent.click(count);
    });
    expect(await screen.findByText(user.name)).toBeTruthy();
  });

  it("should open the users of an enumeration from its menu", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    await openContextMenu(e.name);
    const usedBy = await screen.findByRole("menuitem", { name: "Used by" });
    await act(async () => {
      fireEvent.click(usedBy);
    });
    expect(await screen.findByText("Nothing uses this enumeration.")).toBeTruthy();
  });

  it("should create an enumeration and count up its entry values", async () => {
    await renderEnumerations();
    const name = uniqueName("enumeration");
    const entries = await createThroughForm(name);
    for (let i = 0; i < 2; i++)
      await act(async () => {
        fireEvent.click(within(entries).getByRole("button", { name: "Add entry" }));
      });
    const created = (await client.enumerations.retrieve({ searchTerm: name })).find(
      (x) => x.name === name,
    );
    if (created == null) throw new Error(`the Core holds no ${name}`);
    await expect
      .poll(async () => (await stored(created.key)).map((e) => e.value))
      .toEqual([0, 1]);
  });

  it("should save an entry edited under its enumeration", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const input = await within(await openEntries(e.name)).findByLabelText("Name 2");
    const name = uniqueName("renamed");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: name } });
    fireEvent.blur(input);
    await expect
      .poll(async () => (await stored(e.key)).map((en) => en.name))
      .toEqual([e.entries[0].name, name, e.entries[2].name]);
  });

  it("should show an enumeration's entry count while it is closed", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const item = await screen.findByRole("option", { name: e.name });
    expect(item.getAttribute("aria-expanded")).toBe("false");
    expect(within(item).getByText(String(e.entries.length))).toBeTruthy();
    expect(screen.queryByRole("region", { name: `${e.name} entries` })).toBeNull();
  });

  it("should hide an enumeration's entries when it closes", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    expect(
      await within(await openEntries(e.name)).findByLabelText("Name 1"),
    ).toBeTruthy();
    const item = screen.getByRole("option", { name: e.name });
    expect(item.getAttribute("aria-expanded")).toBe("true");
    fireEvent.click(item);
    expect(screen.queryByRole("region", { name: `${e.name} entries` })).toBeNull();
    expect(item.getAttribute("aria-expanded")).toBe("false");
  });

  it("should open and close an enumeration from the keyboard", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const item = await screen.findByRole("option", { name: e.name });
    expect(item.tabIndex).toBe(0);
    const press = (key: string, expanded: boolean): void => {
      fireEvent.keyDown(item, { key });
      expect(item.getAttribute("aria-expanded"), key).toBe(String(expanded));
    };
    press("Enter", true);
    expect(screen.getByRole("region", { name: `${e.name} entries` })).toBeTruthy();
    press("Enter", false);
    press(" ", true);
    press(" ", false);
    press("ArrowRight", true);
    press("ArrowRight", true);
    press("ArrowLeft", false);
    press("ArrowLeft", false);
    expect(screen.queryByRole("region", { name: `${e.name} entries` })).toBeNull();
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

  it("should keep the users and Replace while the replace runs", async () => {
    const [from, to] = await Promise.all([createEnumeration(), createEnumeration()]);
    const user = await createSelectUser(from.key);
    // While the link is severed, the replace retries, so it stays in flight.
    const { client: editor, proxy } = await createProxiedTestClient({
      retry: { ...FAST_RETRY, maxRetries: 200 },
      onRetry: () => {},
    });
    await renderEnumerations(editor);
    const dialog = await openDelete(from.name);
    expect(
      await within(await dialog.findByLabelText("Users")).findByText(user.name),
    ).toBeTruthy();
    await pickReplacement(dialog, to.name);
    await proxy.sever();
    await clickWhenEnabled(dialog, "Replace");
    expect(dialog.getByRole("button", { name: "Replace" })).toBeTruthy();
    expect(within(dialog.getByLabelText("Users")).getByText(user.name)).toBeTruthy();
    expect(dialog.getByText(/Each of these moves/)).toBeTruthy();
    await proxy.restore();
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
    await waitForDeleted(e.key);
  });

  it("should disable Cancel and Delete while the delete runs", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    await openContextMenu(e.name);
    await clickAndSettle("Delete");
    const remove = await screen.findByRole("button", { name: "Delete" });
    await waitFor(() => expect(remove.getAttribute("aria-disabled")).toBeNull());
    act(() => {
      fireEvent.click(remove);
    });
    expect(remove.getAttribute("aria-disabled")).toBe("true");
    const cancel = screen.getByRole("button", { name: "Cancel" });
    expect(cancel.getAttribute("aria-disabled")).toBe("true");
    await waitForDialogToClose();
  });
});
