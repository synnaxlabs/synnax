// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { color as xcolor, id } from "@synnaxlabs/x";
import { fireEvent, render, type RenderResult, screen } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Color } from "@/color";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const FALLBACK = "#000000";

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

/** Opens the picker of the swatch and waits for the palettes to load into it. */
const openPicker = async (c: RenderResult): Promise<void> => {
  const swatch = c.container.querySelector<HTMLElement>(".lyra-color-swatch");
  if (swatch == null) throw new Error("no swatch rendered");
  fireEvent.click(swatch);
  await screen.findByText("Library");
};

describe("Color.Swatch", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
  });

  it("should call onChange with a named reference for a palette pick", async () => {
    const named = await createInPalette();
    const onChange = vi.fn();
    const c = render(<Color.Swatch fallback={FALLBACK} onChange={onChange} />, {
      wrapper,
    });
    await openPicker(c);
    fireEvent.click(await screen.findByRole("button", { name: named.name }));
    expect(onChange).toHaveBeenCalledWith({ type: "named", key: named.key });
  });

  it("should call onChange with a fixed reference for a custom color", async () => {
    const onChange = vi.fn();
    const c = render(
      <Color.Swatch
        value={{ type: "fixed", value: xcolor.construct("#FF0000") }}
        fallback={FALLBACK}
        onChange={onChange}
      />,
      { wrapper },
    );
    await openPicker(c);
    fireEvent.change(screen.getByLabelText("Hex"), { target: { value: "0000ff" } });
    expect(onChange).toHaveBeenCalledWith({
      type: "fixed",
      value: xcolor.construct("#0000ff"),
    });
  });

  it("should call onChange with undefined when the user picks Auto", async () => {
    const named = await createInPalette();
    const onChange = vi.fn();
    const c = render(
      <Color.Swatch
        value={{ type: "named", key: named.key }}
        fallback={FALLBACK}
        onChange={onChange}
      />,
      { wrapper },
    );
    await openPicker(c);
    fireEvent.click(screen.getByLabelText("Auto"));
    expect(onChange).toHaveBeenCalledExactlyOnceWith(undefined);
  });
});
