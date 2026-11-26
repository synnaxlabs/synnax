// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren, ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Access } from "@/access";
import { Enumeration } from "@/enumeration";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

describe("Enumeration select", () => {
  let wrapper: FC<PropsWithChildren>;
  let created: enumeration.Enumeration[];
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    const prefix = id.create();
    created = await client.enumerations.create([
      { name: `${prefix} b`, entries: [] },
      { name: `${prefix} a`, entries: [] },
    ]);
  });

  const query = (): enumeration.RetrieveMultipleParams => ({
    where: { key: { in: created.map((e) => e.key) } },
  });

  it("should list the enumerations by name and pick one", async () => {
    const [b, a] = created;
    const onChange = vi.fn();
    const c = render(
      <Enumeration.SelectSingle
        value={undefined}
        onChange={onChange}
        query={query()}
        virtual={false}
      />,
      { wrapper },
    );
    fireEvent.click(c.getByRole("button", { name: "Enumeration" }));
    await waitFor(() =>
      expect(c.getAllByRole("option").map((o) => o.textContent)).toEqual([
        a.name,
        b.name,
      ]),
    );
    fireEvent.click(c.getByText(a.name));
    expect(onChange).toHaveBeenCalledWith(a.key, expect.anything());
  });

  it("should leave out the enumerations that the filter rejects", async () => {
    const [b, a] = created;
    const c = render(
      <Enumeration.SelectSingle
        value={undefined}
        onChange={vi.fn()}
        query={query()}
        filter={(e) => e.key !== a.key}
        virtual={false}
      />,
      { wrapper },
    );
    fireEvent.click(c.getByRole("button", { name: "Enumeration" }));
    expect(await c.findByText(b.name)).toBeTruthy();
    expect(c.queryByText(a.name)).toBeNull();
  });

  it("should show the named entries of an enumeration in its option", async () => {
    const e = await client.enumerations.create({
      name: id.create(),
      entries: [
        { key: id.create(), name: "Open", value: 0, abnormal: false },
        { key: id.create(), name: "", value: 1, abnormal: false },
        { key: id.create(), name: "Closed", value: 2, abnormal: true },
      ],
    });
    const c = render(
      <Enumeration.SelectSingle
        value={undefined}
        onChange={vi.fn()}
        query={{ where: { key: { in: [e.key] } } }}
        virtual={false}
      />,
      { wrapper },
    );
    fireEvent.click(c.getByRole("button", { name: "Enumeration" }));
    const option = await c.findByRole("option", { name: e.name });
    expect(option.textContent).toBe(`${e.name}OpenClosed`);
  });

  it("should name the schematics that an edit changes", async () => {
    const e = await client.enumerations.create({ name: id.create(), entries: [] });
    const project = await client.projects.create({ name: id.create() });
    const config = schematic.nodeConfigZ.parse({
      variant: "select",
      enumeration: e.key,
    });
    for (let i = 0; i < 2; i++)
      await client.schematics.create({
        project: project.key,
        name: id.create(),
        nodes: [{ key: "select", position: { x: 0, y: 0 }, zIndex: 0, config }],
      });
    const Harness = (): ReactElement => {
      Access.useEnsurePermissions({});
      return (
        <Enumeration.SelectSingle
          value={undefined}
          onChange={vi.fn()}
          query={{ where: { key: { in: [e.key] } } }}
          virtual={false}
        />
      );
    };
    const c = render(<Harness />, { wrapper });
    fireEvent.click(await c.findByRole("button", { name: "Enumeration" }));
    const option = (await c.findByText(e.name)).closest("[role=option]");
    if (option == null) throw new Error("expected an option");
    fireEvent.contextMenu(option);
    fireEvent.click(await c.findByRole("menuitem", { name: "Edit" }));
    expect(
      await c.findByText("Changes every use of this enumeration, in 2 schematics."),
    ).toBeTruthy();
  });
});
