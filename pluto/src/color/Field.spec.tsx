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
import { color as xcolor, deep, id } from "@synnaxlabs/x";
import {
  fireEvent,
  render,
  type RenderResult,
  screen,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";

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

  const renderField = (fill?: color.Ref, fieldWrapper = wrapper) => {
    let methods!: Form.UseReturn<typeof schema>;
    const Harness = (): ReactElement => {
      methods = Form.use<typeof schema>({ values: { fill }, schema });
      return (
        <Form.Form<typeof schema> {...methods}>
          <Color.Field path={path.fill} label="Fill" fallback="#000000" />
        </Form.Form>
      );
    };
    const result = render(<Harness />, { wrapper: fieldWrapper });
    return { result, values: () => methods.value() };
  };

  const openPicker = (result: RenderResult): void => {
    const swatch = result.container.querySelector<HTMLElement>(".lyra-color-swatch");
    if (swatch == null) throw new Error("no swatch rendered");
    fireEvent.click(swatch);
  };

  const enterName = (placeholder: string, name: string): void => {
    const input = screen.getByPlaceholderText(placeholder);
    fireEvent.change(input, { target: { value: name } });
    fireEvent.keyDown(input, { key: "Enter" });
  };

  it("should store a named reference for a palette pick", async () => {
    const { named } = await createInPalette();
    const { result, values } = renderField();
    openPicker(result);
    fireEvent.click(await screen.findByRole("button", { name: named.name }));
    expect(values().fill).toEqual({ type: "named", key: named.key });
  });

  it("should show the name of a stored named color", async () => {
    const { named } = await createInPalette();
    renderField({ type: "named", key: named.key });
    await waitFor(() =>
      expect(screen.getByRole<HTMLInputElement>("textbox").placeholder).toBe(
        named.name,
      ),
    );
  });

  describe("create and edit", () => {
    it("should create a named color from the shown color and link the field", async () => {
      const { palette } = await createInPalette();
      const { result, values } = renderField();
      openPicker(result);
      fireEvent.click(
        await screen.findByRole("button", {
          name: `Create a color in ${palette.name}`,
        }),
      );
      const name = `created-${id.create()}`;
      enterName("Color name", name);
      await waitFor(() => expect(values().fill?.type).toBe("named"));
      const fill = values().fill;
      if (fill?.type !== "named") throw new Error("the field is not linked");
      const created = await other.colors.retrieve(fill.key);
      expect(created.name).toBe(name);
      expect(created.group).toBe(palette.key);
      expect(xcolor.hex(created.light)).toBe("#000000");
    });

    it("should cancel the create on a blank name", async () => {
      const { palette } = await createInPalette();
      const { result, values } = renderField();
      openPicker(result);
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
      const { result } = renderField();
      openPicker(result);
      await screen.findByRole("button", { name: `Create a color in ${palette.name}` });
      expect(screen.getByRole("group", { name: "Series" })).toBeTruthy();
      expect(
        screen.queryByRole("button", { name: "Create a color in Series" }),
      ).toBeNull();
    });

    it("should save an edit to the linked color from the picker", async () => {
      const { named } = await createInPalette();
      const { result } = renderField({ type: "named", key: named.key });
      openPicker(result);
      fireEvent.click(
        await screen.findByRole("button", { name: `Edit ${named.name}` }),
      );
      fireEvent.click(await screen.findByLabelText("Light theme color"));
      fireEvent.change(screen.getByLabelText("Hex"), { target: { value: "0000ff" } });
      await expect
        .poll(async () => xcolor.hex((await other.colors.retrieve(named.key)).light))
        .toBe("#0000ff");
    });

    it("should create a palette and show it while it has no colors", async () => {
      await createInPalette();
      const { result } = renderField();
      openPicker(result);
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
      const { result } = renderField({ type: "named", key: named.key }, limited);
      openPicker(result);
      // The edit proves the grants answered, so the missing creates are the gate.
      expect(
        await screen.findByRole("button", { name: `Edit ${named.name}` }),
      ).toBeTruthy();
      expect(
        screen.queryByRole("button", { name: `Create a color in ${palette.name}` }),
      ).toBeNull();
      expect(screen.queryByRole("button", { name: "Create palette" })).toBeNull();
    });

    it("should offer no edit to a user who cannot update colors", async () => {
      const { palette, named } = await createInPalette();
      const limited = await createLimitedWrapper("create");
      const { result } = renderField({ type: "named", key: named.key }, limited);
      openPicker(result);
      // The create proves the grants answered, so the missing edit is the gate.
      expect(
        await screen.findByRole("button", {
          name: `Create a color in ${palette.name}`,
        }),
      ).toBeTruthy();
      expect(screen.queryByRole("button", { name: `Edit ${named.name}` })).toBeNull();
    });
  });
});
