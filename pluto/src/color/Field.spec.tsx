// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, color, group, user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { color as xcolor, deep, id } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  type RenderResult,
  screen,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";

import { Access } from "@/access";
import { Color } from "@/color";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const other = createTestClient();

const schema = z.object({ fill: color.refZ.optional() });
const path = deep.path<z.infer<typeof schema>>();

const createInPalette = async (): Promise<{
  palette: group.Group;
  named: color.Named;
}> => {
  const palette = await client.groups.create({
    scope: "color",
    name: `palette-${id.create()}`,
  });
  const named = await client.colors.create({
    name: `color-${id.create()}`,
    light: "#00FF00",
    group: palette.key,
  });
  return { palette, named };
};

describe("Color.Field", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
  });

  const renderField = async (fill?: color.Ref, fieldWrapper = wrapper) => {
    let methods!: Form.UseReturn<typeof schema>;
    const Harness = (): ReactElement => {
      // As in the Console, the field renders once the permissions load.
      Access.useEnsurePermissions({});
      methods = Form.use<typeof schema>({ values: { fill }, schema });
      return (
        <Triggers.Provider>
          <Form.Form<typeof schema> {...methods}>
            <Color.Field path={path.fill} label="Fill" fallback="#000000" />
          </Form.Form>
        </Triggers.Provider>
      );
    };
    let result!: RenderResult;
    // An async act lets a render that suspends on the permissions commit.
    await act(async () => {
      result = render(<Harness />, { wrapper: fieldWrapper });
    });
    return { result, values: () => methods.value() };
  };

  const openPicker = async (result: RenderResult): Promise<void> => {
    const swatch = await waitFor(() => {
      const s = result.container.querySelector<HTMLElement>(".lyra-color-swatch");
      if (s == null) throw new Error("no swatch rendered");
      return s;
    });
    fireEvent.click(swatch);
  };

  const openCreate = async (palette: group.Group): Promise<void> => {
    fireEvent.click(
      await screen.findByRole("button", { name: `Create a color in ${palette.name}` }),
    );
  };

  const createdColor = async (values: () => z.infer<typeof schema>) => {
    await waitFor(() => expect(values().fill?.type).toBe("named"));
    const fill = values().fill;
    if (fill?.type !== "named") throw new Error("the field is not linked");
    return await other.colors.retrieve(fill.key);
  };

  const openSwatchMenu = async (name: string): Promise<void> => {
    fireEvent.contextMenu(await screen.findByRole("button", { name }));
  };

  const openEdit = async (name: string): Promise<void> => {
    await openSwatchMenu(name);
    fireEvent.click(await screen.findByRole("menuitem", { name: "Edit" }));
  };

  const enterName = (placeholder: string, name: string): void => {
    const input = screen.getByPlaceholderText(placeholder);
    fireEvent.change(input, { target: { value: name } });
    fireEvent.keyDown(input, { key: "Enter" });
  };

  it("should store a named reference for a palette pick", async () => {
    const { named } = await createInPalette();
    const { result, values } = await renderField();
    await openPicker(result);
    fireEvent.click(await screen.findByRole("button", { name: named.name }));
    expect(values().fill).toEqual({ type: "named", key: named.key });
  });

  it("should show the name of a stored named color", async () => {
    const { named } = await createInPalette();
    await renderField({ type: "named", key: named.key });
    await waitFor(() =>
      expect(screen.getByRole<HTMLInputElement>("textbox").value).toBe(named.name),
    );
  });

  describe("create and edit", () => {
    it("should create a named color from the shown color and link the field", async () => {
      const { palette } = await createInPalette();
      const { result, values } = await renderField();
      await openPicker(result);
      fireEvent.click(
        await screen.findByRole("button", {
          name: `Create a color in ${palette.name}`,
        }),
      );
      const name = `created-${id.create()}`;
      enterName("Color name", name);
      const created = await createdColor(values);
      expect(created.name).toBe(name);
      expect(created.group).toBe(palette.key);
      expect(xcolor.hex(created.light)).toBe("#000000");
      expect(created.dark).toBeUndefined();
      expect(screen.queryByPlaceholderText("Color name")).toBeNull();
    });

    it("should create one color for a second Enter while the first saves", async () => {
      const { palette } = await createInPalette();
      const { result, values } = await renderField();
      await openPicker(result);
      await openCreate(palette);
      const name = `created-${id.create()}`;
      enterName("Color name", name);
      fireEvent.keyDown(screen.getByPlaceholderText("Color name"), { key: "Enter" });
      await createdColor(values);
      const stored = await other.colors.retrieve({ where: { name: { eq: name } } });
      expect(stored).toHaveLength(1);
    });

    it("should copy both values of the linked color", async () => {
      const { palette } = await createInPalette();
      const linked = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#00FF00",
        dark: "#112233",
        group: palette.key,
      });
      const { result, values } = await renderField({ type: "named", key: linked.key });
      await openPicker(result);
      await openCreate(palette);
      enterName("Color name", `created-${id.create()}`);
      await waitFor(() =>
        expect(values().fill).not.toEqual({ type: "named", key: linked.key }),
      );
      const created = await createdColor(values);
      expect(xcolor.hex(created.light)).toBe("#00ff00");
      expect(created.dark == null ? undefined : xcolor.hex(created.dark)).toBe(
        "#112233",
      );
    });

    it("should keep the shown color in the dark theme", async () => {
      const { palette } = await createInPalette();
      const Outer = wrapper;
      const dark = ({ children }: PropsWithChildren): ReactElement => (
        <Outer>
          <Theming.Provider theme={{ key: "synnaxDark" }} applyCSSVars={false}>
            {children}
          </Theming.Provider>
        </Outer>
      );
      const { result, values } = await renderField(undefined, dark);
      await openPicker(result);
      await openCreate(palette);
      enterName("Color name", `created-${id.create()}`);
      const created = await createdColor(values);
      expect(xcolor.hex(created.light)).toBe("#000000");
      expect(created.dark == null ? undefined : xcolor.hex(created.dark)).toBe(
        "#000000",
      );
    });

    it("should cancel the create on a blank name", async () => {
      const { palette } = await createInPalette();
      const { result, values } = await renderField();
      await openPicker(result);
      fireEvent.click(
        await screen.findByRole("button", {
          name: `Create a color in ${palette.name}`,
        }),
      );
      enterName("Color name", "  ");
      expect(screen.queryByPlaceholderText("Color name")).toBeNull();
      expect(values().fill).toBeUndefined();
    });

    it("should offer no create in the Series palette", async () => {
      const { palette } = await createInPalette();
      const { result } = await renderField();
      await openPicker(result);
      await screen.findByRole("button", { name: `Create a color in ${palette.name}` });
      expect(screen.getByRole("group", { name: "Series" })).toBeTruthy();
      expect(
        screen.queryByRole("button", { name: "Create a color in Series" }),
      ).toBeNull();
    });

    it("should save an edit from the menu of a swatch", async () => {
      const { named } = await createInPalette();
      const { result } = await renderField({ type: "named", key: named.key });
      await openPicker(result);
      await openEdit(named.name);
      fireEvent.click(await screen.findByLabelText("Light theme color"));
      fireEvent.change(screen.getByLabelText("Hex"), { target: { value: "0000ff" } });
      await expect
        .poll(async () => xcolor.hex((await other.colors.retrieve(named.key)).light))
        .toBe("#0000ff");
    });

    it("should keep a rename when Back closes the view before a blur", async () => {
      const { named } = await createInPalette();
      const { result } = await renderField({ type: "named", key: named.key });
      await openPicker(result);
      await openEdit(named.name);
      const name = `renamed-${id.create()}`;
      fireEvent.change(await screen.findByRole("textbox", { name: "Name" }), {
        target: { value: name },
      });
      fireEvent.click(screen.getByLabelText("Back"));
      await expect
        .poll(async () => (await other.colors.retrieve(named.key)).name)
        .toBe(name);
    });

    describe("focus", () => {
      const plusOf = (palette: string): HTMLElement =>
        screen.getByRole("button", { name: `Create a color in ${palette}` });

      it("should move the focus to the new swatch after a create", async () => {
        const { palette } = await createInPalette();
        const { result } = await renderField();
        await openPicker(result);
        await openCreate(palette);
        const name = `created-${id.create()}`;
        enterName("Color name", name);
        const swatch = await screen.findByRole("button", { name });
        await waitFor(() => expect(document.activeElement).toBe(swatch));
      });

      it.each([
        [
          "Escape",
          () =>
            fireEvent.keyDown(screen.getByPlaceholderText("Color name"), {
              key: "Escape",
              code: "Escape",
            }),
        ],
        [
          "Cancel",
          () => fireEvent.click(screen.getByRole("button", { name: "Cancel" })),
        ],
        ["a blank name", () => enterName("Color name", "  ")],
      ])("should keep the picker open and focus the + after %s", async (_, close) => {
        const { palette } = await createInPalette();
        const { result } = await renderField();
        await openPicker(result);
        await openCreate(palette);
        close();
        expect(screen.queryByPlaceholderText("Color name")).toBeNull();
        expect(screen.getByRole("dialog")).toBeTruthy();
        expect(document.activeElement).toBe(plusOf(palette.name));
      });

      it("should move the focus to the + of a new palette", async () => {
        await createInPalette();
        const { result } = await renderField();
        await openPicker(result);
        fireEvent.click(await screen.findByRole("button", { name: "Create palette" }));
        const name = `palette-${id.create()}`;
        enterName("Palette name", name);
        await waitFor(() => expect(document.activeElement).toBe(plusOf(name)));
      });

      const escape = (placeholder: string): void => {
        fireEvent.keyDown(screen.getByPlaceholderText(placeholder), {
          key: "Escape",
          code: "Escape",
        });
      };

      it("should finish a saving color past Escape, Cancel, and +", async () => {
        const { palette } = await createInPalette();
        const { result } = await renderField();
        await openPicker(result);
        await openCreate(palette);
        const name = `created-${id.create()}`;
        enterName("Color name", name);
        escape("Color name");
        fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
        fireEvent.click(plusOf(palette.name));
        expect(screen.getByPlaceholderText("Color name")).toBeTruthy();
        const swatch = await screen.findByRole("button", { name });
        await waitFor(() => expect(document.activeElement).toBe(swatch));
      });

      it("should finish a saving palette past Escape and Cancel", async () => {
        await createInPalette();
        const { result } = await renderField();
        await openPicker(result);
        fireEvent.click(await screen.findByRole("button", { name: "Create palette" }));
        const name = `palette-${id.create()}`;
        enterName("Palette name", name);
        escape("Palette name");
        fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
        expect(screen.getByPlaceholderText("Palette name")).toBeTruthy();
        await waitFor(() => expect(document.activeElement).toBe(plusOf(name)));
      });

      describe("under a search", () => {
        const searchFor = (term: string): void => {
          fireEvent.change(screen.getByPlaceholderText("Search colors"), {
            target: { value: term },
          });
        };
        const searchValue = (): string =>
          screen.getByPlaceholderText<HTMLInputElement>("Search colors").value;

        it("should clear the search and focus a color that it hides", async () => {
          const { palette, named } = await createInPalette();
          const { result } = await renderField();
          await openPicker(result);
          await screen.findByRole("button", { name: named.name });
          searchFor(named.name);
          await openCreate(palette);
          const name = `created-${id.create()}`;
          enterName("Color name", name);
          const swatch = await screen.findByRole("button", { name });
          await waitFor(() => expect(document.activeElement).toBe(swatch));
          expect(searchValue()).toBe("");
        });

        it("should clear the search and focus the + of a hidden palette", async () => {
          const { named } = await createInPalette();
          const { result } = await renderField();
          await openPicker(result);
          await screen.findByRole("button", { name: named.name });
          searchFor(named.name);
          fireEvent.click(screen.getByRole("button", { name: "Create palette" }));
          const name = `palette-${id.create()}`;
          enterName("Palette name", name);
          await waitFor(() => expect(document.activeElement).toBe(plusOf(name)));
          expect(searchValue()).toBe("");
        });

        it("should keep the search after a cancel", async () => {
          const { palette, named } = await createInPalette();
          const { result } = await renderField();
          await openPicker(result);
          await screen.findByRole("button", { name: named.name });
          searchFor(named.name);
          await openCreate(palette);
          fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
          expect(searchValue()).toBe(named.name);
          expect(document.activeElement).toBe(plusOf(palette.name));
        });
      });

      it("should keep the picker open and focus Create palette on Escape", async () => {
        await createInPalette();
        const { result } = await renderField();
        await openPicker(result);
        fireEvent.click(await screen.findByRole("button", { name: "Create palette" }));
        fireEvent.keyDown(screen.getByPlaceholderText("Palette name"), {
          key: "Escape",
          code: "Escape",
        });
        expect(screen.queryByPlaceholderText("Palette name")).toBeNull();
        expect(screen.getByRole("dialog")).toBeTruthy();
        expect(document.activeElement).toBe(
          screen.getByRole("button", { name: "Create palette" }),
        );
      });
    });

    it("should create a palette and show it while it has no colors", async () => {
      await createInPalette();
      const { result } = await renderField();
      await openPicker(result);
      fireEvent.click(await screen.findByRole("button", { name: "Create palette" }));
      const name = `palette-${id.create()}`;
      enterName("Palette name", name);
      expect(await screen.findByRole("group", { name })).toBeTruthy();
    });
  });

  describe("access", () => {
    const createLimitedWrapper = async (
      action: access.Action,
    ): Promise<FC<PropsWithChildren>> => {
      const limited = await createTestClientWithPolicy(client, {
        name: `policy-${id.create()}`,
        objects: [
          color.TYPE_RESOURCE_ID,
          group.TYPE_RESOURCE_ID,
          user.TYPE_RESOURCE_ID,
          access.role.TYPE_RESOURCE_ID,
          access.policy.TYPE_RESOURCE_ID,
        ],
        actions: ["retrieve", action],
      });
      return await createAsyncSynnaxWrapper({ client: limited });
    };

    it("should offer no create to a user who cannot create colors", async () => {
      const { palette, named } = await createInPalette();
      const limited = await createLimitedWrapper("update");
      const { result } = await renderField({ type: "named", key: named.key }, limited);
      await openPicker(result);
      // The edit proves the grants answered, so the missing creates are the gate.
      await openSwatchMenu(named.name);
      expect(await screen.findByRole("menuitem", { name: "Edit" })).toBeTruthy();
      expect(
        screen.queryByRole("button", { name: `Create a color in ${palette.name}` }),
      ).toBeNull();
      expect(screen.queryByRole("button", { name: "Create palette" })).toBeNull();
    });

    it("should open no swatch menu or palette + for a user who cannot update", async () => {
      const { palette, named } = await createInPalette();
      const limited = await createLimitedWrapper("create");
      const { result } = await renderField({ type: "named", key: named.key }, limited);
      await openPicker(result);
      // Create palette proves the grants answered, so the missing actions are the gate.
      expect(
        await screen.findByRole("button", { name: "Create palette" }),
      ).toBeTruthy();
      const swatch = await screen.findByRole("button", { name: named.name });
      act(() => swatch.focus());
      expect(fireEvent.contextMenu(swatch)).toBe(true);
      fireEvent.keyDown(swatch, { key: "F10", code: "F10", shiftKey: true });
      await act(async () => {});
      expect(screen.queryByRole("menuitem", { name: "Edit" })).toBeNull();
      expect(document.activeElement).toBe(swatch);
      expect(
        screen.queryByRole("button", { name: `Create a color in ${palette.name}` }),
      ).toBeNull();
    });

    it("should open the swatch menu only on a color the user can update", async () => {
      const { palette, named } = await createInPalette();
      const locked = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#0000FF",
        group: palette.key,
      });
      const limited = await createTestClientWithPolicy(client, [
        {
          name: `policy-${id.create()}`,
          objects: [
            color.TYPE_RESOURCE_ID,
            group.TYPE_RESOURCE_ID,
            user.TYPE_RESOURCE_ID,
            access.role.TYPE_RESOURCE_ID,
            access.policy.TYPE_RESOURCE_ID,
          ],
          actions: ["retrieve"],
        },
        {
          name: `policy-${id.create()}`,
          objects: [color.resourceID(named.key)],
          actions: ["update"],
        },
      ]);
      const { result } = await renderField(
        { type: "named", key: named.key },
        await createAsyncSynnaxWrapper({ client: limited }),
      );
      await openPicker(result);
      await openSwatchMenu(named.name);
      expect(await screen.findByRole("menuitem", { name: "Edit" })).toBeTruthy();
      fireEvent.keyDown(screen.getByRole("menu"), { key: "Escape", code: "Escape" });
      await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
      const swatch = screen.getByRole("button", { name: locked.name });
      act(() => swatch.focus());
      expect(fireEvent.contextMenu(swatch)).toBe(true);
      fireEvent.keyDown(swatch, { key: "F10", code: "F10", shiftKey: true });
      await act(async () => {});
      expect(screen.queryByRole("menu")).toBeNull();
      expect(document.activeElement).toBe(swatch);
    });

    it("should focus Create palette when a new palette gets no +", async () => {
      const limited = await createLimitedWrapper("create");
      const { result } = await renderField(undefined, limited);
      await openPicker(result);
      fireEvent.click(await screen.findByRole("button", { name: "Create palette" }));
      const name = `palette-${id.create()}`;
      enterName("Palette name", name);
      expect(await screen.findByRole("group", { name })).toBeTruthy();
      await waitFor(() =>
        expect(document.activeElement).toBe(
          screen.getByRole("button", { name: "Create palette" }),
        ),
      );
      expect(
        screen.queryByRole("button", { name: `Create a color in ${name}` }),
      ).toBeNull();
    });

    it("should let a user who cannot update colors create one in no palette", async () => {
      await client.colors.create({ name: `loose-${id.create()}`, light: "#00FF00" });
      const limited = await createLimitedWrapper("create");
      const { result, values } = await renderField(undefined, limited);
      await openPicker(result);
      fireEvent.click(
        await screen.findByRole("button", { name: "Create a color in No palette" }),
      );
      const name = `created-${id.create()}`;
      enterName("Color name", name);
      const created = await createdColor(values);
      expect(created.name).toBe(name);
      expect(created.group).toBeUndefined();
    });
  });
});
