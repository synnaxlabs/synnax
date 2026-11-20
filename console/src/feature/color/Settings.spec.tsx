// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, group, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color as xcolor, compare } from "@synnaxlabs/x";
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
import { describe, expect, it } from "vitest";

import { Color } from "@/feature/color";
import { Modals } from "@/platform/modals";
import {
  awaitGranted,
  awaitTextEditingElement,
  clickAndSettle,
  commitTextEdit,
  createConsoleWrapper,
  createTestClientWithGrants,
  dragOnto,
  HaulProvider,
  openContextMenu,
} from "@/testutil";

const client = createTestClient();

const createPalette = async (): Promise<group.Group> =>
  await client.groups.create({ scope: "color", name: uniqueName("palette") });

const createColor = async (
  palette: group.Group,
  position?: string,
  name: string = uniqueName("color"),
): Promise<color.Named> =>
  await client.colors.create({ name, light: "#00FF00", group: palette.key, position });

/** Returns the nth built-in color from the end, which no other spec here edits. */
const builtinFromEnd = async (n: number): Promise<color.Named> => {
  const shipped = (await client.colors.retrieveBuiltins()).at(-n);
  if (shipped == null) throw new Error("the Core ships no named colors");
  return shipped;
};

const renderColors = async (as: Synnax = client): Promise<void> => {
  const { wrapper: Console } = await createConsoleWrapper({ client: as });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <HaulProvider>{children}</HaulProvider>
      <Modals.Stack />
    </Console>
  );
  await act(async () => {
    render(<Color.Settings />, { wrapper });
  });
};

const deleted = async (retrieve: () => Promise<unknown>): Promise<boolean> => {
  try {
    await retrieve();
    return false;
  } catch (e) {
    return NotFoundError.matches(e);
  }
};

const findColor = async (name: string): Promise<HTMLElement> =>
  await screen.findByRole("option", { name });

/** Names the colors in the palette by position, with any that have none last. */
const namesInOrder = async (palette: group.Group): Promise<string[]> =>
  (await client.colors.retrieve({ groups: [palette.key] }))
    .sort((a, b) => {
      if (a.position === b.position) return compare.text(a.name, b.name);
      if (a.position == null) return 1;
      if (b.position == null) return -1;
      return a.position < b.position ? -1 : 1;
    })
    .map((c) => c.name);

describe("Color.Settings", () => {
  it("should create a palette and open a form for its first color", async () => {
    await renderColors();
    await clickAndSettle("Create palette");
    const name = uniqueName("palette");
    fireEvent.change(await screen.findByLabelText("Palette name"), {
      target: { value: name },
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Create" }));
    });
    expect(await screen.findByText(name)).toBeTruthy();
    expect(await screen.findByPlaceholderText("Color name")).toBeTruthy();
  });

  it("should put new colors at the start of a palette in the order typed", async () => {
    const palette = await createPalette();
    const existing = await createColor(palette, "a0");
    await renderColors();
    const create = await screen.findByRole("button", {
      name: `Create a color in ${palette.name}`,
    });
    await act(async () => {
      fireEvent.click(create);
    });
    const first = uniqueName("first");
    const second = uniqueName("second");
    for (const name of [first, second]) {
      const field = await screen.findByPlaceholderText("Color name");
      fireEvent.change(field, { target: { value: name } });
      await act(async () => {
        fireEvent.keyDown(field, { key: "Enter" });
      });
      await screen.findByText(name);
      // The list can show the color before the save resolves and remounts the form.
      await expect.poll(() => field.isConnected).toBe(false);
    }
    await expect
      .poll(async () => await namesInOrder(palette))
      .toEqual([first, second, existing.name]);
  });

  it("should rename a palette from its menu", async () => {
    const palette = await createPalette();
    await renderColors();
    await openContextMenu(palette.name);
    await clickAndSettle("Rename");
    const name = uniqueName("renamed");
    commitTextEdit(await awaitTextEditingElement(), name);
    await expect
      .poll(async () => (await client.groups.retrieve({ key: palette.key })).name)
      .toBe(name);
  });

  it("should delete a palette and its colors after confirming", async () => {
    const palette = await createPalette();
    const [a, b] = await Promise.all([createColor(palette), createColor(palette)]);
    await renderColors();
    await screen.findByText(a.name);
    await screen.findByText(b.name);
    await openContextMenu(palette.name);
    await clickAndSettle("Delete");
    expect(
      await screen.findByText(
        "This also deletes its 2 colors. This action cannot be undone.",
      ),
    ).toBeTruthy();
    const confirm = await screen.findByRole("button", { name: "Delete" });
    await act(async () => {
      fireEvent.click(confirm);
    });
    await expect
      .poll(
        async () => await deleted(() => client.groups.retrieve({ key: palette.key })),
      )
      .toBe(true);
    await expect
      .poll(async () => await deleted(() => client.colors.retrieve(a.key)))
      .toBe(true);
    await expect
      .poll(async () => await deleted(() => client.colors.retrieve(b.key)))
      .toBe(true);
  });

  it("should not offer to delete the palette of the built-in colors", async () => {
    const builtins = await client.colors.retrieveBuiltins();
    const series = builtins[0].group;
    if (series == null) throw new Error("the built-in colors have no palette");
    // Loads the role's policies, so every grant in the menu answers on first render.
    await awaitGranted(client, group.resourceID(series), "delete");
    await renderColors();
    await screen.findByText(builtins[0].name);
    await openContextMenu("Series");
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should not let a user who cannot create groups create a palette", async () => {
    const limited = await createTestClientWithGrants(client, {
      retrieve: [color.TYPE_RESOURCE_ID, group.TYPE_RESOURCE_ID],
      create: [color.TYPE_RESOURCE_ID],
    });
    await renderColors(limited);
    // The create grant for colors also gates the button, so the "+" proves it answered.
    expect(
      await screen.findByRole("button", { name: "Create a color in Series" }),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Create palette" })).toBeNull();
  });

  it("should picture a dragged color filled with its light value", async () => {
    const shade = await createColor(await createPalette());
    await renderColors();
    fireEvent.dragStart(await findColor(shade.name));
    const layer = document.querySelector(".lyra-haul__layer");
    const tile = layer?.querySelector<HTMLElement>(".lyra-haul-preview__tile");
    expect(tile?.style.background).toBe("rgb(0, 255, 0)");
    expect(layer?.textContent).toBe(`${shade.name}Color`);
  });

  it("should move a dragged color before the one it drops on", async () => {
    const palette = await createPalette();
    const [a, b, c] = await Promise.all([
      createColor(palette, "a0"),
      createColor(palette, "a1"),
      createColor(palette, "a2"),
    ]);
    await renderColors();
    dragOnto(await findColor(c.name), await findColor(a.name));
    await expect
      .poll(async () => await namesInOrder(palette))
      .toEqual([c.name, a.name, b.name]);
    expect((await client.colors.retrieve(a.key)).position).toBe("a0");
  });

  it("should move a color before one that shares a position with another", async () => {
    const palette = await createPalette();
    const a = await createColor(palette, "a0", uniqueName("a"));
    const b = await createColor(palette, "a1", uniqueName("b"));
    const c = await createColor(palette, "a1", uniqueName("c"));
    await renderColors();
    dragOnto(await findColor(a.name), await findColor(c.name));
    await expect
      .poll(async () => await namesInOrder(palette))
      .toEqual([b.name, a.name, c.name]);
  });

  it("should move a color before one with no position", async () => {
    const palette = await createPalette();
    const a = await createColor(palette, "a0", uniqueName("a"));
    const b = await createColor(palette, undefined, uniqueName("b"));
    const c = await createColor(palette, undefined, uniqueName("c"));
    await renderColors();
    dragOnto(await findColor(a.name), await findColor(c.name));
    await expect
      .poll(async () => await namesInOrder(palette))
      .toEqual([b.name, a.name, c.name]);
  });

  it("should mark an edited built-in and restore the values Synnax ships", async () => {
    const shipped = await builtinFromEnd(2);
    await client.colors.retrieve(shipped.key);
    await client.colors.dispatch(
      shipped.key,
      color.set({ path: ["light"], value: xcolor.construct("#123456") }),
    );
    await renderColors();
    expect(
      await within(await findColor(shipped.name)).findByText("Edited"),
    ).toBeTruthy();
    await openContextMenu(shipped.name);
    await clickAndSettle("Restore default");
    await expect
      .poll(async () => xcolor.hex((await client.colors.retrieve(shipped.key)).light))
      .toBe(xcolor.hex(shipped.light));
  });

  it("should leave built-ins out of a delete", async () => {
    const shipped = await builtinFromEnd(3);
    const custom = await createColor(await createPalette());
    await renderColors();
    for (const name of [custom.name, shipped.name])
      fireEvent.click(
        within(await findColor(name)).getByRole("checkbox", { name: "Select" }),
      );
    await openContextMenu(custom.name);
    await clickAndSettle("Delete");
    expect(
      await screen.findByText(`Are you sure you want to delete ${custom.name}?`),
    ).toBeTruthy();
    const confirm = await screen.findByRole("button", { name: "Delete" });
    await act(async () => {
      fireEvent.click(confirm);
    });
    await expect
      .poll(async () => await deleted(() => client.colors.retrieve(custom.key)))
      .toBe(true);
    expect((await client.colors.retrieve(shipped.key)).key).toBe(shipped.key);
  });

  it("should delete every color of a palette that a search hides", async () => {
    const palette = await createPalette();
    const [shown, hidden] = await Promise.all([
      createColor(palette),
      createColor(palette),
    ]);
    await renderColors();
    await findColor(hidden.name);
    fireEvent.change(screen.getByPlaceholderText("Search colors"), {
      target: { value: shown.name },
    });
    await waitFor(() =>
      expect(screen.queryByRole("option", { name: hidden.name })).toBeNull(),
    );
    await openContextMenu(palette.name);
    await clickAndSettle("Delete");
    expect(
      await screen.findByText(
        "This also deletes its 2 colors. This action cannot be undone.",
      ),
    ).toBeTruthy();
    const confirm = await screen.findByRole("button", { name: "Delete" });
    await act(async () => {
      fireEvent.click(confirm);
    });
    await expect
      .poll(async () => await deleted(() => client.colors.retrieve(hidden.key)))
      .toBe(true);
  });

  it("should keep a typed name when another color joins the palette", async () => {
    const palette = await createPalette();
    await createColor(palette, "a0");
    await renderColors();
    fireEvent.click(
      await screen.findByRole("button", { name: `Create a color in ${palette.name}` }),
    );
    const field = await screen.findByPlaceholderText("Color name");
    fireEvent.change(field, { target: { value: "Coolant" } });
    const joined = await createColor(palette, "Zz");
    await findColor(joined.name);
    expect(screen.getByPlaceholderText("Color name")).toHaveProperty(
      "value",
      "Coolant",
    );
  });
});
