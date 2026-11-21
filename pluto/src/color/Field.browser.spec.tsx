// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { deep, id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren, ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";

import { Color } from "@/color";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const schema = z.object({ fill: color.refZ.optional() });
const path = deep.path<z.infer<typeof schema>>();

const createInPalette = async (): Promise<color.Named> => {
  const palette = await client.groups.create({
    scope: "color",
    name: `palette-${id.create()}`,
  });
  return await client.colors.create({
    name: `color-${id.create()}`,
    light: "#00FF00",
    group: palette.key,
  });
};

describe("Color.Field", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
  });

  const renderField = (fill?: color.Ref) => {
    let methods!: Form.UseReturn<typeof schema>;
    const Harness = (): ReactElement => {
      methods = Form.use<typeof schema>({ values: { fill }, schema });
      return (
        <Form.Form<typeof schema> {...methods}>
          <Color.Field path={path.fill} label="Fill" fallback="#000000" />
        </Form.Form>
      );
    };
    const result = render(<Harness />, { wrapper });
    return { result, values: () => methods.value() };
  };

  it("should store a named reference for a palette pick", async () => {
    const named = await createInPalette();
    const { result, values } = renderField();
    const swatch = result.container.querySelector<HTMLElement>(".lyra-color-swatch");
    if (swatch == null) throw new Error("no swatch rendered");
    fireEvent.click(swatch);
    fireEvent.click(await screen.findByRole("button", { name: named.name }));
    expect(values().fill).toEqual({ type: "named", key: named.key });
  });

  it("should show the name of a stored named color", async () => {
    const named = await createInPalette();
    renderField({ type: "named", key: named.key });
    await waitFor(() =>
      expect(screen.getByRole<HTMLInputElement>("textbox").placeholder).toBe(
        named.name,
      ),
    );
  });
});
