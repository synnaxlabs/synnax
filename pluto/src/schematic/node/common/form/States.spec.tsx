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
import { color, deep, id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";
import { type z } from "zod";

import { Form } from "@/schematic/node/common/form";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.StateIndicatorNodeConfig>();

const client = createTestClient();

const findInput = async (label: string): Promise<HTMLInputElement> =>
  await screen.findByLabelText<HTMLInputElement>(label);

const OPTIONS: schematic.StateMapping[] = [
  { key: "off", name: "Off", value: 0 },
  { key: "on", name: "On", value: 1, color: color.construct("#2e7d32") },
];

describe("Form.States", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const renderStates = (
    config: Partial<z.input<typeof schematic.stateIndicatorNodeConfigZ>> = {},
    colorDisabled = false,
  ) =>
    renderForm(
      <Form.States
        enumeration={path.enumeration}
        options={path.options}
        label={path.label.label}
        symbolName="State indicator"
        colorDisabled={colorDisabled}
      />,
      schematic.stateIndicatorNodeConfigZ.parse({
        variant: "state_indicator",
        options: OPTIONS,
        ...config,
      }),
      schematic.stateIndicatorNodeConfigZ,
      wrapper,
    );

  const created = async (
    value: ReturnType<typeof renderStates>["value"],
  ): Promise<enumeration.Enumeration> => {
    let key: enumeration.Key | undefined;
    await waitFor(() => {
      key = value(path.enumeration) as enumeration.Key | undefined;
      expect(key).toBeDefined();
    });
    return await client.enumerations.retrieve(key as enumeration.Key);
  };

  describe("options", () => {
    it("should edit the node's own options when it names no enumeration", async () => {
      const { value } = renderStates();
      const name = await findInput("Name 1");
      fireEvent.focus(name);
      fireEvent.change(name, { target: { value: "Closed" } });
      fireEvent.blur(name);
      await waitFor(() =>
        expect(
          (value(path.options) as schematic.StateMapping[]).map((o) => o.name),
        ).toEqual(["Closed", "On"]),
      );
    });

    it("should keep a named color picked for an option as its color", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const named = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#00FF00",
        group: palette.key,
      });
      const { value } = renderStates();
      fireEvent.click(await screen.findByRole("button", { name: "Pick color 1" }));
      fireEvent.click(await screen.findByRole("button", { name: named.name }));
      await waitFor(() =>
        expect((value(path.options) as schematic.StateMapping[])[0].color).toEqual(
          color.construct("#00FF00"),
        ),
      );
    });

    it("should show empty color cells for a node that draws no color", async () => {
      renderStates({}, true);
      const cell = await findInput("Color 2");
      expect(cell.disabled).toBe(true);
      expect(cell.value).toBe("");
    });
  });

  describe("create", () => {
    const openCreate = async (
      c: ReturnType<typeof renderStates>,
      term = "",
    ): Promise<void> => {
      fireEvent.click(c.getByRole("button", { name: "Enumeration" }));
      if (term !== "")
        fireEvent.change(await c.findByPlaceholderText("Search enumerations..."), {
          target: { value: term },
        });
      fireEvent.click(await c.findByRole("button", { name: "Create enumeration" }));
    };

    it("should name a new enumeration after the search text", async () => {
      const term = `Valve ${id.create()}`;
      const c = renderStates({ label: { label: "Pump" } });
      await openCreate(c, term);
      const e = await created(c.value);
      expect(e.name).toBe(term);
    });

    it("should move the node's options into a new enumeration", async () => {
      const c = renderStates();
      await openCreate(c);
      const e = await created(c.value);
      expect(
        e.entries.map(({ name, value, color: ref }) => ({ name, value, ref })),
      ).toEqual([
        { name: "Off", value: 0, ref: undefined },
        { name: "On", value: 1, ref: { type: "fixed", value: OPTIONS[1].color } },
      ]);
    });

    it("should name a new enumeration after the node's label", async () => {
      const label = `Pump ${id.create()}`;
      const c = renderStates({ label: { label } });
      await openCreate(c);
      expect((await created(c.value)).name).toBe(label);
    });

    it("should name a new enumeration after the symbol with no label", async () => {
      const c = renderStates();
      await openCreate(c);
      expect((await created(c.value)).name).toBe("State indicator");
    });
  });

  describe("enumeration", () => {
    it("should edit the entries of the enumeration that the node names", async () => {
      const e = await client.enumerations.create({
        name: id.create(),
        entries: [{ key: id.create(), name: "Primed", value: 0, abnormal: false }],
      });
      renderStates({ enumeration: e.key });
      expect((await findInput("Name 1")).value).toBe("Primed");
    });

    it("should go back to the node's options on No enumeration", async () => {
      const e = await client.enumerations.create({ name: id.create(), entries: [] });
      const c = renderStates({ enumeration: e.key });
      fireEvent.click(await c.findByRole("button", { name: /^Enumeration/ }));
      fireEvent.click(await c.findByText("No enumeration"));
      await waitFor(() => expect(c.value(path.enumeration)).toBeUndefined());
      expect((await findInput("Name 1")).value).toBe("Off");
    });
  });
});
