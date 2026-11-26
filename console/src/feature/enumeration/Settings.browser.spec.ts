// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration, schematic } from "@synnaxlabs/client";
import { createTestClientWithPolicy, holdResponses } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { assert, describe, expect, it, onTestFinished } from "vitest";
import { server } from "vitest/browser";

import {
  clickWhenEnabled,
  client,
  core,
  createEnumeration,
  createSelectUser,
  dialogOf,
  openDelete,
  pickReplacement,
  renderEnumerations,
  waitForDialogToClose,
} from "@/feature/enumeration/testutil";
import { findButton } from "@/platform/modals/testutil";
import {
  awaitGranted,
  awaitTextEditingElement,
  clickAndSettle,
  commitTextEdit,
  openContextMenu,
} from "@/testutil";

// Chromium and WebKit can run at once against one Core, so each edits its own built-in.
const EDITED_FROM_END = server.browser === "webkit" ? 4 : 2;

/** Returns the nth built-in from the end, which no other spec edits. */
const builtinFromEnd = async (n: number): Promise<enumeration.Enumeration> => {
  const shipped = (await client.enumerations.retrieveBuiltins()).at(-n);
  if (shipped == null) throw new Error("the Core ships no enumerations");
  return shipped;
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

  it("should open a new enumeration with its first entry focused", async () => {
    await renderEnumerations();
    const name = uniqueName("enumeration");
    const entries = await createThroughForm(name);
    await waitFor(() =>
      expect(document.activeElement).toBe(within(entries).getByLabelText("Name 1")),
    );
    expect(screen.getByRole("option", { name }).getAttribute("aria-expanded")).toBe(
      "true",
    );
    await act(async () => {
      fireEvent.click(within(entries).getByRole("button", { name: "Add entry" }));
    });
    await waitFor(() =>
      expect(document.activeElement).toBe(within(entries).getByLabelText("Name 2")),
    );
  });

  it("should list a new enumeration first before the Core answers", async () => {
    const prefix = uniqueName("order");
    await client.enumerations.create({ name: `${prefix}-a` });
    await renderEnumerations();
    fireEvent.change(screen.getByPlaceholderText("Search enumerations"), {
      target: { value: prefix },
    });
    await waitFor(() => expect(screen.getAllByRole("option")).toHaveLength(1));
    const { release } = holdResponses(client, "/enumeration/create");
    await clickAndSettle("Create enumeration");
    const created = `${prefix}-z`;
    fireEvent.change(await screen.findByLabelText("Enumeration name"), {
      target: { value: created },
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Create" }));
    });
    const optionNames = (): Array<string | null> =>
      screen.getAllByRole("option").map((o) => o.getAttribute("aria-label"));
    await waitFor(() => expect(optionNames()).toEqual([created, `${prefix}-a`]));
    release();
    // The new enumeration opens once the Core answers.
    await screen.findByRole("region", { name: `${created} entries` });
    expect(optionNames()).toEqual([created, `${prefix}-a`]);
  });

  it("should keep focus in the search when a new enumeration shows again", async () => {
    await renderEnumerations();
    const name = uniqueName("enumeration");
    const entries = await createThroughForm(name);
    await waitFor(() =>
      expect(document.activeElement).toBe(within(entries).getByLabelText("Name 1")),
    );
    const search = screen.getByPlaceholderText("Search enumerations");
    act(() => search.focus());
    // One word: the Core lists the partial matches of a term with several words.
    fireEvent.change(search, { target: { value: id.create() } });
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
    // One word: the Core lists the partial matches of a term with several words.
    fireEvent.change(search, { target: { value: id.create() } });
    // A create before the debounced search lands would show the new enumeration.
    await screen.findByText("No enumerations");
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
    await waitFor(() =>
      expect(document.activeElement).toBe(within(first).getByLabelText("Name 1")),
    );
    const second = await createThroughForm(uniqueName("enumeration"));
    await waitFor(() =>
      expect(document.activeElement).toBe(within(second).getByLabelText("Name 1")),
    );
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
      .toEqual([0, 1, 2]);
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

  it("should keep the settings menu off a right click in the entries", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const input = await within(await openEntries(e.name)).findByLabelText("Name 1");
    // A menu that opens prevents the default, which hides the browser's own menu.
    expect(fireEvent.contextMenu(input)).toBe(true);
    await openContextMenu(e.name);
    expect(await screen.findByText("Rename")).toBeTruthy();
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
    const shipped = await builtinFromEnd(EDITED_FROM_END);
    await client.enumerations.retrieve(shipped.key);
    await client.enumerations.dispatch(
      shipped.key,
      enumeration.set({ path: ["entries"], value: shipped.entries.slice(1) }),
    );
    // A failed restore must not leave the built-in edited for later runs.
    onTestFinished(async () => {
      await client.enumerations.dispatch(
        shipped.key,
        enumeration.set({ path: ["entries"], value: shipped.entries }),
      );
    });
    await renderEnumerations();
    // The other browser's run can mark its own built-in edited on the same page.
    const heading = (): HTMLElement => {
      const el = screen.getByText(shipped.name).closest<HTMLElement>("[aria-expanded]");
      assert(el != null);
      return el;
    };
    await waitFor(() => expect(within(heading()).getByText("Edited")).toBeTruthy());
    await openBuiltinMenu(shipped.name);
    await clickAndSettle("Restore default");
    await expect.poll(async () => await stored(shipped.key)).toEqual(shipped.entries);
    await waitFor(() => expect(within(heading()).queryByText("Edited")).toBeNull());
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

  it("should not name a hidden schematic in a refused delete", async () => {
    const e = await createEnumeration();
    await createSelectUser(e.key);
    const limited = await createTestClientWithPolicy(client, {
      name: uniqueName("policy"),
      objects: [enumeration.TYPE_RESOURCE_ID],
      actions: ["retrieve", "delete"],
    });
    await awaitGranted(limited, enumeration.TYPE_RESOURCE_ID, "delete");
    const { statuses } = await renderEnumerations(limited);
    const dialog = await openDelete(e.name);
    expect(await dialog.findByText(/Nothing uses it/)).toBeTruthy();
    await clickWhenEnabled(dialog, "Delete");
    await waitFor(() =>
      expect(statuses()).toContainEqual(
        expect.objectContaining({
          message: "Failed to delete enumeration",
          description:
            `cannot delete enumeration ${e.name} because the schematic that uses it ` +
            "as its nodes.config.enumeration is not visible to you; ask someone who " +
            "can see it to remove the use",
        }),
      ),
    );
  });

  it("should keep the users and Replace after the Core refuses a replace", async () => {
    const e = await createEnumeration();
    const to = await createEnumeration();
    const user = await createSelectUser(e.key);
    const limited = await createTestClientWithPolicy(client, [
      {
        name: uniqueName("policy"),
        objects: [enumeration.TYPE_RESOURCE_ID],
        actions: ["retrieve", "delete"],
      },
      {
        name: uniqueName("policy"),
        objects: [schematic.TYPE_RESOURCE_ID],
        actions: ["retrieve"],
      },
    ]);
    await awaitGranted(limited, enumeration.TYPE_RESOURCE_ID, "delete");
    const { statuses } = await renderEnumerations(limited);
    const dialog = await openDelete(e.name);
    expect(await dialog.findByText(user.name)).toBeTruthy();
    await pickReplacement(dialog, to.name);
    await clickWhenEnabled(dialog, "Replace");
    await waitFor(() =>
      expect(statuses()).toContainEqual(
        expect.objectContaining({ message: "Failed to replace enumeration" }),
      ),
    );
    expect(dialog.getByText(user.name)).toBeTruthy();
    expect(dialog.getByRole("button", { name: "Replace" })).toBeTruthy();
    expect(dialog.queryByText(/Nothing uses it/)).toBeNull();
  });

  it("should keep the users and Replace while the replace runs", async () => {
    const [from, to] = await Promise.all([createEnumeration(), createEnumeration()]);
    const user = await createSelectUser(from.key);
    await renderEnumerations();
    const dialog = await openDelete(from.name);
    expect(
      await within(await dialog.findByLabelText("Users")).findByText(user.name),
    ).toBeTruthy();
    await pickReplacement(dialog, to.name);
    // The replace stays in flight until the Core's answer is released.
    const { release } = holdResponses(client, "/enumeration/replace");
    await clickWhenEnabled(dialog, "Replace");
    expect(dialog.getByRole("button", { name: "Replace" })).toBeTruthy();
    expect(within(dialog.getByLabelText("Users")).getByText(user.name)).toBeTruthy();
    expect(dialog.getByText(/Each of these moves/)).toBeTruthy();
    release();
    await waitForDialogToClose();
    await expect
      .poll(async () => (await core.schematics.retrieve(user.key)).nodes[0].config)
      .toMatchObject({ variant: "select", enumeration: to.key });
  });

  it("should delete every checked enumeration from the menu of one", async () => {
    const checked = [await createEnumeration(), await createEnumeration()];
    await renderEnumerations();
    for (const { name } of checked) {
      const option = await screen.findByRole("option", { name });
      fireEvent.click(within(option).getByRole("checkbox", { name: "Select" }));
    }
    await openContextMenu(checked[1].name);
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByRole("menuitem", { name: /Used by/ })).toBeNull();
    await clickAndSettle("Delete");
    fireEvent.click(findButton("Delete"));
    for (const { key } of checked) await waitForDeleted(key);
  });

  it("should offer no Delete when one checked enumeration is in use", async () => {
    const checked = [await createEnumeration(), await createEnumeration()];
    await createSelectUser(checked[0].key);
    await awaitGranted(client, enumeration.resourceID(checked[0].key), "delete");
    await renderEnumerations();
    const used = await screen.findByRole("option", { name: checked[0].name });
    await within(used).findByText("Used by 1");
    for (const { name } of checked) {
      const option = await screen.findByRole("option", { name });
      fireEvent.click(within(option).getByRole("checkbox", { name: "Select" }));
    }
    await openContextMenu(checked[1].name);
    expect(screen.queryByRole("menuitem", { name: /Used by/ })).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
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
