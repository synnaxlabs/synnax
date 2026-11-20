// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, enumeration } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color as xcolor } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Enumeration } from "@/feature/enumeration";
import { Modals } from "@/platform/modals";
import {
  awaitGranted,
  awaitTextEditingElement,
  CaptureStatuses,
  clickAndSettle,
  commitTextEdit,
  createConsoleWrapper,
  dragOnto,
  HaulProvider,
  openContextMenu,
} from "@/testutil";

const client = createTestClient();
// Reads what the Core holds, without the edits that the cache of client shows at once.
const core = createTestClient();

const createEnumeration = async (
  entries: enumeration.Entry[] = [
    { key: "a", value: 0, name: uniqueName("a"), abnormal: false },
    { key: "b", value: 1, name: uniqueName("b"), abnormal: false },
    { key: "c", value: 2, name: uniqueName("c"), abnormal: false },
  ],
): Promise<enumeration.Enumeration> =>
  await client.enumerations.create({ name: uniqueName("enumeration"), entries });

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
      <HaulProvider>{children}</HaulProvider>
      <Modals.Stack />
      <CaptureStatuses onStatuses={onStatuses} />
    </Console>
  );
  await act(async () => {
    render(<Enumeration.Settings />, { wrapper });
  });
};

const findEntry = async (name: string): Promise<HTMLElement> =>
  await screen.findByRole("option", { name });

const stored = async (key: enumeration.Key): Promise<enumeration.Entry[]> =>
  (await core.enumerations.retrieve(key)).entries;

const storedNames = async (key: enumeration.Key): Promise<string[]> =>
  (await stored(key)).map((e) => e.name);

// The delete dialog enables its button once it knows what uses the enumeration.
const clickWhenEnabled = async (name: string): Promise<void> => {
  const button = await screen.findByRole("button", { name });
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

const OWN_VALUE = "Each entry needs its own value.";

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
    expect(await screen.findByText(name)).toBeTruthy();
    for (const entry of ["Shut", "Ajar"]) {
      const input = await screen.findByPlaceholderText("Entry name");
      fireEvent.change(input, { target: { value: entry } });
      await act(async () => {
        fireEvent.keyDown(input, { key: "Enter", code: "Enter" });
      });
    }
    const created = (await client.enumerations.retrieve({ searchTerm: name })).find(
      (x) => x.name === name,
    );
    if (created == null) throw new Error(`the Core holds no ${name}`);
    await expect
      .poll(async () => (await stored(created.key)).map((e) => [e.value, e.name]))
      .toEqual([
        [0, "Shut"],
        [1, "Ajar"],
      ]);
  });

  it("should rename an entry from its menu", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    await openContextMenu(e.entries[1].name);
    await clickAndSettle("Rename");
    const name = uniqueName("renamed");
    commitTextEdit(await awaitTextEditingElement(), name);
    await expect
      .poll(async () => await storedNames(e.key))
      .toEqual([e.entries[0].name, name, e.entries[2].name]);
  });

  it("should store an entry as abnormal", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const entry = await findEntry(e.entries[2].name);
    fireEvent.click(within(entry).getByLabelText("Abnormal"));
    await expect
      .poll(async () => (await stored(e.key)).map((x) => x.abnormal))
      .toEqual([false, false, true]);
  });

  it("should delete each selected entry from the menu", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    for (const entry of [e.entries[0], e.entries[2]])
      fireEvent.click(
        within(await findEntry(entry.name)).getByRole("checkbox", { name: "Select" }),
      );
    await openContextMenu(e.entries[2].name);
    await clickAndSettle("Remove");
    await expect
      .poll(async () => await storedNames(e.key))
      .toEqual([e.entries[1].name]);
  });

  it("should offer no edits for entries selected in two enumerations", async () => {
    const [a, b] = await Promise.all([createEnumeration(), createEnumeration()]);
    await awaitGranted(client, enumeration.resourceID(b.key), "update");
    await renderEnumerations();
    await openContextMenu(b.entries[0].name);
    expect(await screen.findByText("Remove")).toBeTruthy();
    fireEvent.keyDown(document.body, { key: "Escape" });
    for (const entry of [a.entries[0], b.entries[0]])
      fireEvent.click(
        within(await findEntry(entry.name)).getByRole("checkbox", { name: "Select" }),
      );
    await openContextMenu(b.entries[0].name);
    expect(await screen.findByText("Reload Console")).toBeTruthy();
    expect(screen.queryByText("Remove")).toBeNull();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should store a value typed into an entry", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const input = within(await findEntry(e.entries[1].name)).getByLabelText("Value");
    fireEvent.change(input, { target: { value: "7" } });
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" });
    await expect
      .poll(async () => (await stored(e.key)).map((x) => x.value))
      .toEqual([0, 7, 2]);
  });

  it.each([
    { holder: "named", name: uniqueName("c"), shown: (n: string) => n },
    { holder: "unnamed", name: "", shown: () => "Another entry" },
  ])(
    "should refuse a value that a $holder entry holds and say so",
    async ({ name, shown }) => {
      const e = await createEnumeration([
        { key: "a", value: 0, name: uniqueName("a"), abnormal: false },
        { key: "b", value: 1, name: uniqueName("b"), abnormal: false },
        { key: "c", value: 2, name, abnormal: false },
      ]);
      const onStatuses = vi.fn();
      await renderEnumerations(onStatuses);
      const input = within(await findEntry(e.entries[1].name)).getByLabelText("Value");
      fireEvent.change(input, { target: { value: "2" } });
      fireEvent.keyDown(input, { key: "Enter", code: "Enter" });
      await waitFor(() =>
        expect(onStatuses).toHaveBeenLastCalledWith([
          expect.objectContaining({
            message: "Value in use",
            description: `${shown(name)} has the value 2. ${OWN_VALUE}`,
          }),
        ]),
      );
      await act(async () => {});
      expect((await stored(e.key)).map((x) => x.value)).toEqual([0, 1, 2]);
    },
  );

  it("should add an entry with the value typed into the add form", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const add = await screen.findByRole("button", {
      name: `Add an entry to ${e.name}`,
    });
    await act(async () => {
      fireEvent.click(add);
    });
    const value = await screen.findByLabelText("Entry value");
    fireEvent.change(value, { target: { value: "9" } });
    fireEvent.keyDown(value, { key: "Enter", code: "Enter" });
    const name = screen.getByLabelText("Entry name");
    fireEvent.change(name, { target: { value: "Tripped" } });
    await act(async () => {
      fireEvent.keyDown(name, { key: "Enter" });
    });
    await expect
      .poll(async () => (await stored(e.key)).map((x) => [x.value, x.name]))
      .toContainEqual([9, "Tripped"]);
  });

  it("should suggest a value above every entry after one fills a gap", async () => {
    const e = await createEnumeration([
      { key: "a", value: 0, name: uniqueName("a"), abnormal: false },
      { key: "b", value: 1, name: uniqueName("b"), abnormal: false },
      { key: "c", value: 5, name: uniqueName("c"), abnormal: false },
    ]);
    await renderEnumerations();
    const add = await screen.findByRole("button", {
      name: `Add an entry to ${e.name}`,
    });
    await act(async () => {
      fireEvent.click(add);
    });
    const value = await screen.findByLabelText<HTMLInputElement>("Entry value");
    fireEvent.change(value, { target: { value: "2" } });
    fireEvent.keyDown(value, { key: "Enter", code: "Enter" });
    const name = screen.getByLabelText("Entry name");
    fireEvent.change(name, { target: { value: "Ajar" } });
    await act(async () => {
      fireEvent.keyDown(name, { key: "Enter" });
    });
    await waitFor(() => expect(value.value).toBe("6"));
  });

  it("should store a picked entry color and clear it with Auto", async () => {
    const e = await createEnumeration();
    await renderEnumerations();
    const entry = await findEntry(e.entries[0].name);
    fireEvent.click(within(entry).getByLabelText("Entry color"));
    fireEvent.change(await screen.findByLabelText("Hex"), {
      target: { value: "ff0000" },
    });
    await expect
      .poll(async () => (await stored(e.key))[0].color)
      .toEqual({ type: "fixed", value: xcolor.construct("#ff0000") });
    fireEvent.click(screen.getByLabelText("Auto"));
    await expect.poll(async () => (await stored(e.key))[0].color).toBeUndefined();
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

  it("should move a dragged entry before the one it drops on", async () => {
    const e = await createEnumeration();
    const [a, b, c] = e.entries;
    await renderEnumerations();
    dragOnto(await findEntry(c.name), await findEntry(a.name));
    await expect
      .poll(async () => await storedNames(e.key))
      .toEqual([c.name, a.name, b.name]);
  });

  it("should picture a dragged entry filled with its color", async () => {
    const e = await createEnumeration([
      {
        key: "on",
        value: 1,
        name: uniqueName("on"),
        abnormal: false,
        color: { type: "fixed", value: xcolor.construct("#00FF00") },
      },
    ]);
    await renderEnumerations();
    fireEvent.dragStart(await findEntry(e.entries[0].name));
    const layer = document.querySelector(".lyra-haul__layer");
    const tile = layer?.querySelector<HTMLElement>(".lyra-haul-preview__tile");
    expect(tile?.style.background).toBe("rgb(0, 255, 0)");
    expect(layer?.textContent).toBe(`${e.entries[0].name}Value 1`);
  });

  it("should take a dragged entry only over entries of its own enumeration", async () => {
    const [e, other] = await Promise.all([createEnumeration(), createEnumeration()]);
    await renderEnumerations();
    const dragged = await findEntry(e.entries[0].name);
    const sibling = await findEntry(e.entries[1].name);
    const stranger = await findEntry(other.entries[1].name);
    const over = "console-enumeration-settings__item--over";
    fireEvent.dragStart(dragged);
    fireEvent.dragOver(stranger, { screenX: 1, screenY: 1 });
    fireEvent.dragOver(sibling, { screenX: 2, screenY: 2 });
    expect(sibling.className).toContain(over);
    expect(stranger.className).not.toContain(over);
    fireEvent.dragEnd(dragged);
  });

  it("should warn about each entry that is hard to tell apart", async () => {
    const twin = uniqueName("twin");
    const [white, black, green, greenish, red] = [
      "white",
      "black",
      "green",
      "greenish",
      "red",
    ].map(uniqueName);
    const fixed = (hex: string): color.Ref => ({
      type: "fixed",
      value: xcolor.construct(hex),
    });
    const entry = (value: number, name: string, c?: color.Ref): enumeration.Entry => ({
      key: `e${value}`,
      value,
      name,
      abnormal: false,
      color: c,
    });
    // A fixed color adapts to the dark theme, so only a dark value can fail there.
    const night = await client.colors.create({
      name: uniqueName("night"),
      light: "#2E9E4F",
      dark: "#0A0A0A",
    });
    const e = await createEnumeration([
      entry(0, twin),
      entry(1, twin),
      entry(2, ""),
      entry(3, white, fixed("#FDFDF0")),
      entry(4, green, fixed("#2E9E4F")),
      entry(5, greenish, fixed("#2F9F50")),
      entry(6, red, fixed("#D7263D")),
      entry(7, black, { type: "named", key: night.key }),
    ]);
    await renderEnumerations();
    // Other specs leave entries with no name, so a search shows only this enumeration.
    fireEvent.change(screen.getByPlaceholderText("Search enumerations"), {
      target: { value: e.name },
    });
    await waitFor(() =>
      expect(screen.getAllByLabelText(/It has no name/)).toHaveLength(1),
    );
    const warns = async (name: string, message: RegExp): Promise<void> => {
      const options = await screen.findAllByRole("option", { name });
      expect(within(options[0]).getByLabelText(message)).toBeTruthy();
    };
    await warns(twin, /Another entry has its name/);
    await warns(white, /hard to see in the light theme/);
    await warns(black, /hard to see in the dark theme/);
    expect(within(await findEntry(black)).getByText(night.name)).toBeTruthy();
    await warns(green, new RegExp(`almost the color of ${greenish}`));
    await warns(red, new RegExp(`With color blindness, its color looks like ${green}`));
    await warns(red, /It has an alarm color, but it is not abnormal/);
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
    await openContextMenu(shipped.name);
    await clickAndSettle("Restore default");
    await expect.poll(async () => await stored(shipped.key)).toEqual(shipped.entries);
    await waitFor(() => expect(screen.queryByText("Edited")).toBeNull());
  });

  it("should not offer to delete a built-in", async () => {
    const shipped = await builtinFromEnd(3);
    await awaitGranted(client, enumeration.resourceID(shipped.key), "delete");
    await renderEnumerations();
    await openContextMenu(shipped.name);
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.queryByText("Delete")).toBeNull();
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
    expect(screen.getByText(/This action cannot be undone/)).toBeTruthy();
    expect(screen.queryByText(/Nothing uses it/)).toBeNull();
    expect(await screen.findByText(/Nothing uses it/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Replace with/ })).toBeNull();
    await clickWhenEnabled("Delete");
    await waitForDialogToClose();
    await expect(client.enumerations.retrieve(e.key)).rejects.toThrow(
      "enumeration was deleted",
    );
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
