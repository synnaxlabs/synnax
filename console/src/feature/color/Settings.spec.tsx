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
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
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
  openContextMenu,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createPalette = async (): Promise<group.Group> =>
  await client.groups.create({ scope: "color", name: uniqueName("palette") });

const createColor = async (
  palette: group.Group,
  position?: string,
): Promise<color.Named> =>
  await client.colors.create({
    name: uniqueName("color"),
    light: "#00FF00",
    group: palette.key,
    position,
  });

const renderColors = async (as: Synnax = client): Promise<void> => {
  const { wrapper: Console } = await createConsoleWrapper({ client: as });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      {children}
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

/** Names the colors in the palette by position, as the Core orders them. */
const namesInOrder = async (palette: group.Group): Promise<string[]> =>
  (await client.colors.retrieve({ groups: [palette.key] }))
    .sort((a, b) => ((a.position ?? "") < (b.position ?? "") ? -1 : 1))
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
    await act(async () => {
      fireEvent.click(
        await screen.findByRole("button", {
          name: `Create a color in ${palette.name}`,
        }),
      );
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
    await act(async () => {
      fireEvent.click(await screen.findByRole("button", { name: "Delete" }));
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
});
