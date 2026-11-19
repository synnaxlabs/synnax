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
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

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
    render(<Label.SelectMultiple value={[lbl.key]} onChange={vi.fn()} />, { wrapper });
    const expected = color.cssString("#123456");
    const tag = await screen.findByText(lbl.name);
    await waitFor(() => expect(dotColor(tag.closest(".lyra-tag"))).toBe(expected));
    fireEvent.click(tag);
    fireEvent.change(await screen.findByPlaceholderText("Search labels..."), {
      target: { value: lbl.name },
    });
    const option = await screen.findByRole("option", { name: lbl.name });
    await waitFor(() => expect(dotColor(option)).toBe(expected));
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
    await act(async () => {
      render(<Harness />, { wrapper });
    });
    fireEvent.click(await screen.findByText(lbl.name));
    fireEvent.change(await screen.findByPlaceholderText("Search labels..."), {
      target: { value: lbl.name },
    });
    const option = await screen.findByRole("option", { name: lbl.name });
    expect(fireEvent.contextMenu(option)).toBe(false);
    fireEvent.click(await screen.findByText("Edit"));
    const name = await screen.findByLabelText("Label name");
    fireEvent.change(name, { target: { value: `${lbl.name}-renamed` } });
    await waitFor(async () =>
      expect((await client.labels.retrieve(lbl.key)).name).toBe(`${lbl.name}-renamed`),
    );
  });
});
