// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { color, id } from "@synnaxlabs/x";
import { fireEvent } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";
import { render } from "vitest-browser-react";

import { Access } from "@/access";
import { Label } from "@/label";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const dotColor = (element: Element | null): string | null | undefined =>
  element?.querySelector("svg")?.getAttribute("color");

describe("Label.SelectMultiple", () => {
  it("should paint its tags and options in the named color of the label", async () => {
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const named = await client.colors.create({
      name: `select-${id.create()}`,
      light: "#123456",
    });
    const lbl = await client.labels.create({
      name: `named-${id.create()}`,
      color: { type: "named", key: named.key },
    });
    await render(<Label.SelectMultiple value={[lbl.key]} onChange={vi.fn()} />, {
      wrapper,
    });
    const expected = color.cssString("#123456");
    const tag = page.getByText(lbl.name);
    await expect.element(tag).toBeInTheDocument();
    await expect
      .poll(() => dotColor(tag.element().closest(".lyra-tag")))
      .toBe(expected);
    await tag.click();
    await userEvent.fill(page.getByPlaceholder("Search labels..."), lbl.name);
    const option = page.getByRole("option", { name: lbl.name });
    await expect.element(option).toBeInTheDocument();
    await expect.poll(() => dotColor(option.element())).toBe(expected);
  });
});

describe("Label.SelectSingle", () => {
  it("should edit the label of an option from its context menu", async () => {
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const lbl = await client.labels.create({
      name: `edit-${id.create()}`,
      color: { type: "fixed", value: "#123456" },
    });
    const Harness = (): ReactElement => {
      // As in the Console, the select renders once the permissions load.
      Access.useEnsurePermissions({});
      return <Label.SelectSingle value={lbl.key} onChange={vi.fn()} virtual={false} />;
    };
    await render(<Harness />, { wrapper });
    await page.getByText(lbl.name).click();
    await userEvent.fill(page.getByPlaceholder("Search labels..."), lbl.name);
    const option = page.getByRole("option", { name: lbl.name });
    await expect.element(option).toBeInTheDocument();
    expect(fireEvent.contextMenu(option.element())).toBe(false);
    await page.getByText("Edit").click();
    await userEvent.fill(page.getByLabelText("Label name"), `${lbl.name}-renamed`);
    await expect
      .poll(async () => (await client.labels.retrieve(lbl.key)).name)
      .toBe(`${lbl.name}-renamed`);
  });
});
