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
import { CreatePromptStub } from "@/schematic/node/testutil";
import {
  createAsyncSynnaxWrapper,
  createSynnaxWrapper,
  renderForm,
  renderFormSuspended,
} from "@/testutil";

const path = deep.path<schematic.SymbolNodeConfig>();

const commit = (input: HTMLElement, value: string): void => {
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const VALVE: schematic.symbol.Ref = { type: "builtin", name: "valve" };

describe("Symbol form", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  let heater: schematic.symbol.Ref;
  let ch: channel.Channel;
  beforeAll(async () => {
    const Base = await createAsyncSynnaxWrapper({ client });
    wrapper = ({ children }) => (
      <Base>
        <CreatePromptStub>{children}</CreatePromptStub>
      </Base>
    );
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
    heater = { type: "custom", key: symbol.key };
  });

  const render = async (
    config: Partial<schematic.SymbolNodeConfig>,
    tab: "telemetry" | "control" = "telemetry",
  ) =>
    await renderFormSuspended(
      <Form tab={tab} onTabChange={vi.fn()} />,
      schematic.symbolNodeConfigZ.parse({ variant: "symbol", ...config }),
      schematic.symbolNodeConfigZ,
      wrapper,
    );

  describe("sources", () => {
    it.each([
      ["Channel", { type: "channel" }],
      ["Local", { type: "local", value: 0 }],
    ])(
      "should add a binding on the first pick of a %s source",
      async (label, source) => {
        const { findByText, value } = await render({ spec: VALVE });
        fireEvent.click(await findByText(label));
        expect(value(path.states)).toEqual([
          schematic.stateBindingZ.parse({ key: "active", source }),
        ]);
      },
    );

    it("should keep the binding when its source is cleared", async () => {
      const { findByText, value } = await render({
        spec: VALVE,
        states: [
          schematic.stateBindingZ.parse({ key: "active", source: { type: "channel" } }),
        ],
      });
      fireEvent.click(await findByText("None"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active" }),
      ]);
    });
  });

  describe("overrides", () => {
    it("should replace the match of an enumeration state", async () => {
      const { findByText, value } = await render({ spec: VALVE });
      fireEvent.click(await findByText("Exact"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", match: { type: "exact" } }),
      ]);
    });

    it("should keep the bounds of a within match on a click of Within", async () => {
      const states = [
        schematic.stateBindingZ.parse({
          key: "active",
          match: { type: "within", bounds: { lower: 2, upper: 3 } },
        }),
      ];
      const { findByText, value } = await render({ spec: VALVE, states });
      fireEvent.click(await findByText("Within"));
      expect(value(path.states)).toEqual(states);
    });

    it("should replace the bounds of a within match", async () => {
      const { findByLabelText, value } = await render({ spec: VALVE });
      commit(await findByLabelText("Max value"), "1.5");
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({
          key: "active",
          match: { type: "within", bounds: { lower: 0.9, upper: 1.5 } },
        }),
      ]);
    });

    it("should start a within match at the bounds of the symbol", async () => {
      const { findByText, value } = await render({
        spec: heater,
        states: [
          schematic.stateBindingZ.parse({ key: "active", match: { type: "exact" } }),
        ],
      });
      fireEvent.click(await findByText("Within"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({
          key: "active",
          match: { type: "within", bounds: { lower: 2, upper: 3 } },
        }),
      ]);
    });

    const within = (lower: number, upper: number) =>
      schematic.stateBindingZ.parse({
        key: "active",
        match: { type: "within", bounds: { lower, upper } },
      });
    const INVALID = "Min value must be less than max value";

    it("should start a within match at 0.9 to 1.1 on an exact symbol", async () => {
      const exact = await client.schematics.symbols.create({
        name: "Lamp",
        data: schematic.symbol.specZ.parse({
          drawing: { type: "svg", markup: "<svg/>" },
          variant: "static",
          states: [{ type: "enumeration", key: "active", name: "Active" }],
        }),
      });
      const { findByText, value } = await render({
        spec: { type: "custom", key: exact.key },
      });
      fireEvent.click(await findByText("Within"));
      expect(value(path.states)).toEqual([within(0.9, 1.1)]);
    });

    it("should hold a minimum above the maximum until the pair is valid", async () => {
      const { findByLabelText, getByLabelText, getByText, value } = await render({
        spec: VALVE,
      });
      commit(await findByLabelText("Max value"), "100");
      commit(getByLabelText("Min value"), "200");
      expect(getByText(INVALID)).toBeDefined();
      expect(value(path.states)).toEqual([within(0.9, 100)]);
      commit(getByLabelText("Max value"), "300");
      expect(value(path.states)).toEqual([within(200, 300)]);
    });

    it("should hold a minimum equal to the maximum and save nothing", async () => {
      const { findByLabelText, getByText, value } = await render({ spec: VALVE });
      commit(await findByLabelText("Min value"), "1.1");
      expect(getByText(INVALID)).toBeDefined();
      expect(value(path.states)).toEqual([]);
    });

    it("should show the error on the field that the user edited", async () => {
      const { findByLabelText, getByLabelText } = await render({ spec: VALVE });
      commit(await findByLabelText("Max value"), "0.5");
      expect(getByLabelText("Max value").getAttribute("aria-invalid")).toBe("true");
      expect(getByLabelText("Min value").getAttribute("aria-invalid")).toBeNull();
    });

    it("should drop a held pair when the stored bounds change", async () => {
      const { findByLabelText, getByLabelText, getByText, queryByText } = await render({
        spec: VALVE,
        states: [within(2, 3)],
      });
      commit(await findByLabelText("Min value"), "5");
      fireEvent.click(getByText("Reset to symbol"));
      expect(queryByText(INVALID)).toBeNull();
      expect(getByLabelText("Min value")).toHaveProperty("value", "0.9");
    });

    it("should replace the default of a state with no enumeration", async () => {
      const { findByLabelText, value } = await render({ spec: heater });
      commit(await findByLabelText("Default"), "1");
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", default: 1 }),
      ]);
    });

    it("should replace the bounds of a number state", async () => {
      const { findAllByLabelText, value } = await render({ spec: heater });
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

    it("should clear the overrides and keep the source on a reset", async () => {
      const source = { type: "channel" } as const;
      const { findByText, value } = await render({
        spec: VALVE,
        states: [
          schematic.stateBindingZ.parse({
            key: "active",
            source,
            match: { type: "exact" },
            default: 1,
          }),
        ],
      });
      fireEvent.click(await findByText("Reset to symbol"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", source }),
      ]);
    });
  });

  it("should warn when no entry has the value of the default", async () => {
    const { findByText } = await render({
      spec: VALVE,
      states: [schematic.stateBindingZ.parse({ key: "active", default: 5 })],
    });
    expect(await findByText("No entry has the value 5.")).toBeDefined();
  });

  it("should tell the user when the symbol names no enumeration", async () => {
    const { findByText } = await render({ spec: heater });
    expect(await findByText("The symbol names no enumeration.")).toBeDefined();
  });

  it.each(["telemetry", "control"] as const)(
    "should warn on the %s tab when the symbol is missing",
    (tab) => {
      const { getByText } = renderForm(
        <Form tab={tab} onTabChange={vi.fn()} />,
        schematic.symbolNodeConfigZ.parse({
          variant: "symbol",
          spec: { type: "missing", key: id.create() },
        }),
        schematic.symbolNodeConfigZ,
        createSynnaxWrapper({ client: null }),
      );
      expect(getByText("The symbol of this node was not found.")).toBeDefined();
    },
  );

  it.each(["telemetry", "control"] as const)(
    "should warn on the %s tab when the symbol fails to load",
    async (tab) => {
      // A user with no policies cannot read the symbol, so its load fails.
      const denied = await createTestClientWithPolicy(client, []);
      const { findByText } = renderForm(
        <Form tab={tab} onTabChange={vi.fn()} />,
        schematic.symbolNodeConfigZ.parse({ variant: "symbol", spec: heater }),
        schematic.symbolNodeConfigZ,
        await createAsyncSynnaxWrapper({ client: denied }),
      );
      expect(await findByText("The symbol of this node failed to load.")).toBeDefined();
    },
  );

  describe("commands", () => {
    it("should list only the commands with no state", async () => {
      const { findByText, queryByText } = await render({ spec: heater }, "control");
      await findByText("Reset");
      expect(queryByText("On")).toBeNull();
    });

    it("should remove a command binding when its channel is cleared", async () => {
      const { findByLabelText, value } = await render(
        { spec: heater, commands: [{ key: "reset", channel: ch.key }] },
        "control",
      );
      fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
      await waitFor(() => expect(value(path.commands)).toEqual([]));
    });
  });

  it("should store no units when the units are cleared", async () => {
    const { findByLabelText, value } = await render({
      spec: heater,
      states: [schematic.stateBindingZ.parse({ key: "level", units: "psi" })],
    });
    fireEvent.change(await findByLabelText("Units"), { target: { value: "" } });
    expect(value(path.states)).toEqual([
      schematic.stateBindingZ.parse({ key: "level" }),
    ]);
  });

  it("should show the format of a number state only", async () => {
    const { findByText, getAllByLabelText } = await render({ spec: heater });
    await findByText("Level");
    expect(getAllByLabelText("Units")).toHaveLength(1);
  });
});
