// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, schematic } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { deep, id } from "@synnaxlabs/x";
import { fireEvent, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Form } from "@/schematic/node/symbol/Form";
import { createAsyncSynnaxWrapper, createSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.SymbolNodeConfig>();

const parse = (
  config: Partial<schematic.SymbolNodeConfig>,
): schematic.SymbolNodeConfig =>
  schematic.symbolNodeConfigZ.parse({ variant: "symbol", ...config });

const commit = (input: HTMLElement, value: string): void => {
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

describe("Symbol form", () => {
  describe("built-in symbol", () => {
    const valve: schematic.symbol.Ref = { type: "builtin", name: "valve" };

    it.each([
      ["Channel", { type: "channel" }],
      ["Local", { type: "local", value: 0 }],
    ])("should add a binding on the first pick of a %s source", (label, source) => {
      const { getByText, value } = renderForm(
        <Form tab="telemetry" onTabChange={vi.fn()} />,
        parse({ spec: valve }),
        schematic.symbolNodeConfigZ,
        createSynnaxWrapper({ client: null }),
      );
      fireEvent.click(getByText(label));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", source }),
      ]);
    });

    it("should keep the binding when its source is cleared", () => {
      const { getByText, value } = renderForm(
        <Form tab="telemetry" onTabChange={vi.fn()} />,
        parse({
          spec: valve,
          states: [
            schematic.stateBindingZ.parse({
              key: "active",
              source: { type: "channel" },
            }),
          ],
        }),
        schematic.symbolNodeConfigZ,
        createSynnaxWrapper({ client: null }),
      );
      fireEvent.click(getByText("None"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active" }),
      ]);
    });
  });

  describe("overrides", () => {
    const valve: schematic.symbol.Ref = { type: "builtin", name: "valve" };
    const render = (states: schematic.StateBinding[] = []) =>
      renderForm(
        <Form tab="telemetry" onTabChange={vi.fn()} />,
        parse({ spec: valve, states }),
        schematic.symbolNodeConfigZ,
        createSynnaxWrapper({ client: null }),
      );

    it("should replace the match of an enumeration state", () => {
      const { getByText, value } = render();
      fireEvent.click(getByText("Exact"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", match: { type: "exact" } }),
      ]);
    });

    it("should keep the bounds of a within match on a click of Within", () => {
      const states = [
        schematic.stateBindingZ.parse({
          key: "active",
          match: { type: "within", bounds: { lower: 2, upper: 3 } },
        }),
      ];
      const { getByText, value } = render(states);
      fireEvent.click(getByText("Within"));
      expect(value(path.states)).toEqual(states);
    });

    it("should replace the bounds of a within match", () => {
      const { getByLabelText, value } = render();
      commit(getByLabelText("Max value"), "1.5");
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({
          key: "active",
          match: { type: "within", bounds: { lower: 0.9, upper: 1.5 } },
        }),
      ]);
    });

    const within = (lower: number, upper: number) =>
      schematic.stateBindingZ.parse({
        key: "active",
        match: { type: "within", bounds: { lower, upper } },
      });
    const INVALID = "Min value must be less than max value";

    it("should hold a minimum above the maximum until the pair is valid", () => {
      const { getByLabelText, getByText, value } = render();
      commit(getByLabelText("Max value"), "100");
      commit(getByLabelText("Min value"), "200");
      expect(getByText(INVALID)).toBeDefined();
      expect(value(path.states)).toEqual([within(0.9, 100)]);
      commit(getByLabelText("Max value"), "300");
      expect(value(path.states)).toEqual([within(200, 300)]);
    });

    it("should hold a minimum equal to the maximum and save nothing", () => {
      const { getByLabelText, getByText, value } = render();
      commit(getByLabelText("Min value"), "1.1");
      expect(getByText(INVALID)).toBeDefined();
      expect(value(path.states)).toEqual([]);
    });

    it("should show the error on the field that the user edited", () => {
      const { getByLabelText } = render();
      commit(getByLabelText("Max value"), "0.5");
      expect(getByLabelText("Max value").getAttribute("aria-invalid")).toBe("true");
      expect(getByLabelText("Min value").getAttribute("aria-invalid")).toBeNull();
    });

    it("should drop a held pair when the stored bounds change", () => {
      const { getByLabelText, getByText, queryByText } = render([within(2, 3)]);
      commit(getByLabelText("Min value"), "5");
      fireEvent.click(getByText("Reset to symbol"));
      expect(queryByText(INVALID)).toBeNull();
      expect(getByLabelText("Min value")).toHaveProperty("value", "0.9");
    });

    it("should replace the default of an enumeration state", () => {
      const { getByLabelText, value } = render();
      commit(getByLabelText("Default"), "1");
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", default: 1 }),
      ]);
    });

    it("should clear the overrides and keep the source on a reset", () => {
      const source = { type: "channel" } as const;
      const { getByText, value } = render([
        schematic.stateBindingZ.parse({
          key: "active",
          source,
          match: { type: "exact" },
          default: 1,
        }),
      ]);
      fireEvent.click(getByText("Reset to symbol"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", source }),
      ]);
    });
  });

  it.each(["telemetry", "control"] as const)(
    "should warn on the %s tab when the symbol is missing",
    (tab) => {
      const { getByText } = renderForm(
        <Form tab={tab} onTabChange={vi.fn()} />,
        parse({ spec: { type: "missing", key: id.create() } }),
        schematic.symbolNodeConfigZ,
        createSynnaxWrapper({ client: null }),
      );
      expect(getByText("The symbol of this node was not found.")).toBeDefined();
    },
  );

  describe("custom symbol", () => {
    const client = createTestClient();
    let wrapper: FC<PropsWithChildren>;
    let ref: schematic.symbol.Ref;
    let ch: channel.Channel;
    beforeAll(async () => {
      wrapper = await createAsyncSynnaxWrapper({ client });
      ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const symbol = await client.schematics.symbols.create({
        name: "Heater",
        data: schematic.symbol.specZ.parse({
          drawing: { type: "svg", markup: "<svg/>" },
          variant: "static",
          states: [
            {
              type: "enumeration",
              key: "active",
              name: "Active",
              match: { type: "within", bounds: { lower: 2, upper: 3 } },
            },
            { type: "number", key: "level", name: "Level" },
          ],
          commands: [
            {
              key: "on",
              name: "On",
              state: "active",
              value: { type: "fixed", value: 1 },
            },
            { key: "reset", name: "Reset", value: { type: "fixed", value: 0 } },
          ],
        }),
      });
      ref = { type: "custom", key: symbol.key };
    });

    it.each(["telemetry", "control"] as const)(
      "should warn on the %s tab when the symbol fails to load",
      async (tab) => {
        // A user with no policies cannot read the symbol, so its load fails.
        const denied = await createTestClientWithPolicy(client, []);
        const { findByText } = renderForm(
          <Form tab={tab} onTabChange={vi.fn()} />,
          parse({ spec: ref }),
          schematic.symbolNodeConfigZ,
          await createAsyncSynnaxWrapper({ client: denied }),
        );
        expect(
          await findByText("The symbol of this node failed to load."),
        ).toBeDefined();
      },
    );

    it("should store no units when the units are cleared", async () => {
      const { findByLabelText, value } = renderForm(
        <Form tab="telemetry" onTabChange={vi.fn()} />,
        parse({
          spec: ref,
          states: [schematic.stateBindingZ.parse({ key: "level", units: "psi" })],
        }),
        schematic.symbolNodeConfigZ,
        wrapper,
      );
      fireEvent.change(await findByLabelText("Units"), { target: { value: "" } });
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "level" }),
      ]);
    });

    it("should show the format of a number state only", async () => {
      const { findByText, getAllByLabelText } = renderForm(
        <Form tab="telemetry" onTabChange={vi.fn()} />,
        parse({ spec: ref }),
        schematic.symbolNodeConfigZ,
        wrapper,
      );
      await findByText("Level");
      expect(getAllByLabelText("Units")).toHaveLength(1);
    });

    it("should replace the bounds of a number state", async () => {
      const { findAllByLabelText, value } = renderForm(
        <Form tab="telemetry" onTabChange={vi.fn()} />,
        parse({ spec: ref }),
        schematic.symbolNodeConfigZ,
        wrapper,
      );
      // The within match of the Active state comes first.
      const [, level] = await findAllByLabelText("Max value");
      commit(level, "50");
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({
          key: "level",
          bounds: { lower: 0, upper: 50 },
        }),
      ]);
    });

    it("should start a within match at the bounds of the symbol", async () => {
      const { findByText, value } = renderForm(
        <Form tab="telemetry" onTabChange={vi.fn()} />,
        parse({
          spec: ref,
          states: [
            schematic.stateBindingZ.parse({ key: "active", match: { type: "exact" } }),
          ],
        }),
        schematic.symbolNodeConfigZ,
        wrapper,
      );
      fireEvent.click(await findByText("Within"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({
          key: "active",
          match: { type: "within", bounds: { lower: 2, upper: 3 } },
        }),
      ]);
    });

    it("should start a within match at 0.9 to 1.1 on an exact symbol", async () => {
      const exact = await client.schematics.symbols.create({
        name: "Lamp",
        data: schematic.symbol.specZ.parse({
          drawing: { type: "svg", markup: "<svg/>" },
          variant: "static",
          states: [{ type: "enumeration", key: "active", name: "Active" }],
        }),
      });
      const { findByText, value } = renderForm(
        <Form tab="telemetry" onTabChange={vi.fn()} />,
        parse({ spec: { type: "custom", key: exact.key } }),
        schematic.symbolNodeConfigZ,
        wrapper,
      );
      fireEvent.click(await findByText("Within"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({
          key: "active",
          match: { type: "within", bounds: { lower: 0.9, upper: 1.1 } },
        }),
      ]);
    });

    it("should list only the commands with no state", async () => {
      const { findByText, queryByText } = renderForm(
        <Form tab="control" onTabChange={vi.fn()} />,
        parse({ spec: ref }),
        schematic.symbolNodeConfigZ,
        wrapper,
      );
      await findByText("Reset");
      expect(queryByText("On")).toBeNull();
    });

    it("should remove a command binding when its channel is cleared", async () => {
      const { findByLabelText, value } = renderForm(
        <Form tab="control" onTabChange={vi.fn()} />,
        parse({ spec: ref, commands: [{ key: "reset", channel: ch.key }] }),
        schematic.symbolNodeConfigZ,
        wrapper,
      );
      fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
      await waitFor(() => expect(value(path.commands)).toEqual([]));
    });
  });
});
