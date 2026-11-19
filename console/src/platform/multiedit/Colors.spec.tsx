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
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { MultiEdit } from "@/platform/multiedit";
import { getHexInput, getSwatches } from "@/platform/multiedit/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const FALLBACK = "#000000";
const GREEN = "#00ff00";

const createInPalette = async (): Promise<color.Named> => {
  const palette = await client.groups.create({
    scope: "color",
    name: uniqueName("palette"),
  });
  return await client.colors.create({
    name: uniqueName("color"),
    light: GREEN,
    group: palette.key,
  });
};

const named = (key: string): color.Ref => ({ type: "named", key });

describe("ColorField", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    ({ wrapper } = await createConsoleWrapper({ client }));
    await waitForStreamLive(client.connection);
  });

  it("should send a named reference for a palette pick", async () => {
    const c = await createInPalette();
    const onChange = vi.fn();
    render(
      <MultiEdit.ColorField
        label="Fill"
        values={[{ fallback: FALLBACK }]}
        onChange={onChange}
      />,
      { wrapper },
    );
    fireEvent.click(getSwatches("Fill")[0]);
    fireEvent.click(await screen.findByRole("button", { name: c.name }));
    expect(onChange).toHaveBeenCalledWith(named(c.key));
  });

  it("should show the name of the named color that every element follows", async () => {
    const c = await createInPalette();
    render(
      <MultiEdit.ColorField
        label="Fill"
        values={[
          { value: named(c.key), fallback: FALLBACK },
          { value: named(c.key), fallback: FALLBACK },
        ]}
        onChange={vi.fn()}
      />,
      { wrapper },
    );
    await waitFor(() => expect(getHexInput("Fill").placeholder).toBe(c.name));
  });

  it("should show Mixed when one element follows a named color and another holds its value", async () => {
    const c = await createInPalette();
    render(
      <MultiEdit.ColorField
        label="Fill"
        values={[
          { value: named(c.key), fallback: FALLBACK },
          { value: MultiEdit.fixedRef(GREEN), fallback: FALLBACK },
        ]}
        onChange={vi.fn()}
      />,
      { wrapper },
    );
    await waitFor(() => expect(getHexInput("Fill").placeholder).toBe("Mixed"));
  });

  it("should hide the palettes when palettesHidden is set", async () => {
    const c = await createInPalette();
    render(
      <MultiEdit.ColorField
        label="Fill"
        values={[{ value: named(c.key), fallback: FALLBACK }]}
        onChange={vi.fn()}
        palettesHidden
      />,
      { wrapper },
    );
    // The named color resolves once the named colors load.
    await waitFor(() => expect(getHexInput("Fill").value).toBe(GREEN.slice(1)));
    fireEvent.click(getSwatches("Fill")[0]);
    await screen.findByLabelText("Hex");
    expect(screen.queryByText("Library")).toBeNull();
  });
});

describe("SelectionColors", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    ({ wrapper } = await createConsoleWrapper({ client }));
    await waitForStreamLive(client.connection);
  });

  it("should group holders by named color, apart from fixed colors", async () => {
    const c = await createInPalette();
    render(
      <MultiEdit.SelectionColors
        refs={[
          { key: "a", path: "fillColor", value: named(c.key) },
          { key: "b", path: "fillColor", value: named(c.key) },
          { key: "c", path: "fillColor", value: MultiEdit.fixedRef(GREEN) },
        ]}
        onChange={vi.fn()}
      />,
      { wrapper },
    );
    await waitFor(() => expect(getSwatches("Selection")).toHaveLength(2));
  });

  it("should recolor every holder of a group with a named reference", async () => {
    const c = await createInPalette();
    const onChange = vi.fn();
    const refs: MultiEdit.ColorRef[] = [
      { key: "a", path: "fillColor", value: MultiEdit.fixedRef(GREEN) },
      { key: "b", path: "textColor", value: MultiEdit.fixedRef(GREEN) },
    ];
    render(<MultiEdit.SelectionColors refs={refs} onChange={onChange} />, { wrapper });
    fireEvent.click(getSwatches("Selection")[0]);
    // A fixed color opens the picker on its Custom tab.
    fireEvent.click(await screen.findByText("Library"));
    fireEvent.click(await screen.findByRole("button", { name: c.name }));
    expect(onChange).toHaveBeenCalledWith(refs, named(c.key));
  });
});
