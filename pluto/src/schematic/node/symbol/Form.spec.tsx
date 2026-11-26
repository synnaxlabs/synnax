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
const LIGHT: schematic.symbol.Ref = { type: "builtin", name: "light" };

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
    it("should add a binding on the first pick of a Local source", async () => {
      const { findByText, value } = await render({ spec: VALVE });
      fireEvent.click(await findByText("Local"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({
          key: "active",
          source: { type: "local", value: 0 },
        }),
      ]);
    });

    it("should restore the earlier channel on a switch back from Local", async () => {
      const source = { type: "channel", key: ch.key, command: ch.key } as const;
      const { findByText, getByText, value } = await render({
        spec: VALVE,
        states: [schematic.stateBindingZ.parse({ key: "active", source })],
      });
      fireEvent.click(await findByText("Local"));
      fireEvent.click(getByText("Channel"));
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", source }),
      ]);
    });

    it("should keep the binding when its channel is cleared", async () => {
      const { findByLabelText, value } = await render({
        spec: VALVE,
        states: [
          schematic.stateBindingZ.parse({
            key: "active",
            source: { type: "channel", key: ch.key },
          }),
        ],
      });
      fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
      await waitFor(() =>
        expect(value(path.states)).toEqual([
          schematic.stateBindingZ.parse({
            key: "active",
            source: { type: "channel" },
          }),
        ]),
      );
    });
  });

  describe("overrides", () => {
    const within = (lower: number, upper: number) =>
      schematic.stateBindingZ.parse({
        key: "active",
        match: { type: "within", bounds: { lower, upper } },
      });

    it("should replace the threshold of a within match", async () => {
      const { findByLabelText, value } = await render({ spec: LIGHT });
      commit(await findByLabelText("Threshold end"), "1.5");
      expect(value(path.states)).toEqual([within(0.9, 1.5)]);
    });

    it("should show no threshold for a state that a command writes", async () => {
      const { findByText, queryByLabelText } = await render({ spec: VALVE });
      await findByText("States");
      expect(queryByLabelText("Threshold start")).toBeNull();
    });

    it("should drop a held pair when the stored bounds change", async () => {
      const { container, findByLabelText, getByLabelText, getByText } = await render({
        spec: LIGHT,
        states: [within(2, 3)],
      });
      commit(await findByLabelText("Threshold start"), "5");
      expect(container.querySelector(".lyra--error")).not.toBeNull();
      fireEvent.click(getByText("Reset"));
      expect(container.querySelector(".lyra--error")).toBeNull();
      expect(getByLabelText("Threshold start")).toHaveProperty("value", "0.9");
    });

    it("should replace the default of a state with no enumeration", async () => {
      const { findByLabelText, value } = await render({ spec: heater });
      commit(await findByLabelText("Default"), "1");
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", default: 1 }),
      ]);
    });

    it("should replace the range of a number state", async () => {
      const { findByLabelText, value } = await render({ spec: heater });
      commit(await findByLabelText("Range end"), "50");
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
      fireEvent.click(await findByText("Reset"));
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

  it.each(["Telemetry", "Control"] as const)(
    "should warn on the %s tab when the symbol is missing",
    (name) => {
      const tab = name === "Control" ? "control" : "telemetry";
      const { getByRole, getByText } = renderForm(
        <Form tab={tab} onTabChange={vi.fn()} />,
        schematic.symbolNodeConfigZ.parse({
          variant: "symbol",
          spec: { type: "missing", key: id.create() },
        }),
        schematic.symbolNodeConfigZ,
        createSynnaxWrapper({ client: null }),
      );
      expect(getByRole("tab", { name, selected: true })).toBeDefined();
      expect(getByText("The symbol of this node was not found.")).toBeDefined();
    },
  );

  it.each(["Telemetry", "Control"] as const)(
    "should warn on the %s tab when the symbol fails to load",
    async (name) => {
      const tab = name === "Control" ? "control" : "telemetry";
      // A user with no policies cannot read the symbol, so its load fails.
      const denied = await createTestClientWithPolicy(client, []);
      const { findByRole, findByText } = renderForm(
        <Form tab={tab} onTabChange={vi.fn()} />,
        schematic.symbolNodeConfigZ.parse({ variant: "symbol", spec: heater }),
        schematic.symbolNodeConfigZ,
        await createAsyncSynnaxWrapper({ client: denied }),
      );
      expect(await findByText("The symbol of this node failed to load.")).toBeDefined();
      expect(await findByRole("tab", { name, selected: true })).toBeDefined();
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

  describe("control tab", () => {
    it("should show no Control tab for a symbol with no click or command", async () => {
      const { findByRole, queryByRole } = await render({ spec: LIGHT });
      await findByRole("tab", { name: "Style" });
      expect(queryByRole("tab", { name: "Control" })).toBeNull();
    });

    it("should show the clicks of a symbol with a click", async () => {
      const { findByLabelText } = await render({ spec: VALVE }, "control");
      expect(await findByLabelText("Clickable")).toBeDefined();
    });

    it("should show the chip but no clicks of a symbol with no click", async () => {
      const { findByLabelText, queryByLabelText } = await render(
        { spec: heater },
        "control",
      );
      expect(await findByLabelText("Control chip")).toBeDefined();
      expect(queryByLabelText("Clickable")).toBeNull();
    });

    it("should keep the Control tab when every command has a state", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "Lamp",
        data: schematic.symbol.specZ.parse({
          drawing: { type: "svg", markup: "<svg/>" },
          states: [{ type: "enumeration", key: "active", name: "Active" }],
          commands: [
            {
              key: "on",
              name: "On",
              state: "active",
              value: { type: "fixed", value: 1 },
            },
          ],
        }),
      });
      const { findByLabelText } = await render(
        { spec: { type: "custom", key: symbol.key } },
        "control",
      );
      expect(await findByLabelText("Control chip")).toBeDefined();
    });

    it("should open the Telemetry tab of a light when Control was open", async () => {
      const { findByRole, findByText } = await render({ spec: LIGHT }, "control");
      await findByRole("tab", { name: "Telemetry", selected: true });
      expect(await findByText("Staleness")).toBeDefined();
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
