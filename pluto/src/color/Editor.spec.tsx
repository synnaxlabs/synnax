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
import { color as xcolor, id } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Color } from "@/color";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const other = createTestClient();

describe("Color.Editor", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
  });

  const renderEditor = (key: color.Key, disabled?: boolean) =>
    render(
      <Color.Editor itemKey={key}>
        <Color.LightInput disabled={disabled} />
        <Color.DarkInput disabled={disabled} />
      </Color.Editor>,
      { wrapper },
    );

  const storedHex = async (
    key: color.Key,
    field: "light" | "dark",
  ): Promise<string | undefined> => {
    const value = (await other.colors.retrieve(key))[field];
    return value == null ? undefined : xcolor.hex(value);
  };

  it("should save a picked light value to the Core", async () => {
    const named = await client.colors.create({
      name: `editor-${id.create()}`,
      light: "#00ff00",
    });
    const c = renderEditor(named.key);
    fireEvent.click(await c.findByLabelText("Light theme color"));
    fireEvent.change(c.getByLabelText("Hex"), { target: { value: "0000ff" } });
    await expect.poll(async () => await storedHex(named.key, "light")).toBe("#0000ff");
  });

  it("should show Auto for an absent dark value until one is picked", async () => {
    const named = await client.colors.create({
      name: `editor-${id.create()}`,
      light: "#00ff00",
    });
    const c = renderEditor(named.key);
    expect(await c.findByText("Auto")).toBeTruthy();
    fireEvent.click(c.getByLabelText("Dark theme color"));
    fireEvent.change(c.getByLabelText("Hex"), { target: { value: "ff0000" } });
    await expect.poll(async () => await storedHex(named.key, "dark")).toBe("#ff0000");
    expect(c.queryByText("Auto")).toBeNull();
  });

  it("should show an edit that another client makes", async () => {
    const named = await client.colors.create({
      name: `editor-${id.create()}`,
      light: "#00ff00",
    });
    const c = renderEditor(named.key);
    await c.findByText("#00FF00");
    await other.colors.retrieve(named.key);
    await other.colors.dispatch(named.key, [
      color.set({ path: ["light"], value: xcolor.construct("#123456") }),
    ]);
    expect(await c.findByText("#123456")).toBeTruthy();
  });

  it("should offer no picker while disabled", async () => {
    const named = await client.colors.create({
      name: `editor-${id.create()}`,
      light: "#00ff00",
    });
    const c = renderEditor(named.key, true);
    fireEvent.click(await c.findByLabelText("Light theme color"));
    fireEvent.click(c.getByLabelText("Dark theme color"));
    expect(c.queryByLabelText("Hex")).toBeNull();
  });
});
