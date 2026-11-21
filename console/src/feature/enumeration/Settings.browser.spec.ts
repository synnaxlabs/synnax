// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration, schematic } from "@synnaxlabs/client";
import { createTestClientWithPolicy } from "@synnaxlabs/client/testutil";
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

  it("should keep the settings menu off a right click in the entries", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const input = await within(await findSection(e.name)).findByLabelText("Name 1");
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
    await expect(client.enumerations.retrieve(e.key)).rejects.toThrow(
      "enumeration was deleted",
    );
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
