// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type project, type schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

const builtIn = (name: string): Schematic.Node.Input<"symbol"> => ({
  variant: "symbol",
  spec: { type: "builtin", name },
});

describe("useChangeSymbol", () => {
  let Wrapper: FC<PropsWithChildren>;
  let proj: project.Project;
  beforeAll(async () => {
    [Wrapper, proj] = await Promise.all([
      createAsyncSynnaxWrapper({ client }),
      client.projects.create({ name: `project_${uuid.create()}` }),
    ]);
  });

  // Renders useChangeSymbol over a cached schematic that holds the node as n1.
  const renderChange = async (config: schematic.NodeConfig) => {
    const { key } = await client.schematics.create({
      project: proj.key,
      name: `schematic_${uuid.create()}`,
      nodes: [{ key: "n1", position: { x: 0, y: 0 }, config }],
    });
    await client.schematics.retrieve(key);
    const Scoped: FC<PropsWithChildren> = ({ children }) => (
      <Wrapper>
        <Schematic.Scope.Provider value={key}>{children}</Schematic.Scope.Provider>
      </Wrapper>
    );
    const { result } = renderHook(() => Schematic.useChangeSymbol(), {
      wrapper: Scoped,
    });
    const configOf = async () =>
      client.schematics.configsOf(await client.schematics.retrieve(key)).n1;
    return { change: result.current, configOf };
  };

  it("should keep the channels of an old node changed to a built-in", async () => {
    const { change, configOf } = await renderChange(
      Schematic.Node.createConfig({
        variant: "valve",
        stateChannel: 12,
        commandChannel: 13,
      }),
    );
    act(() => change(["n1"], builtIn("solenoid_valve")));
    await waitFor(async () =>
      expect(await configOf()).toMatchObject({
        variant: "solenoid_valve",
        stateChannel: 12,
        commandChannel: 13,
      }),
    );
  });

  it("should change an old node to the old node of a custom symbol", async () => {
    const symbol = await client.schematics.symbols.create({
      name: `symbol_${uuid.create()}`,
      data: {
        drawing: { type: "svg", markup: "<svg></svg>" },
        states: [],
        handles: [],
        variant: "actuator",
        scale: 1,
        strokeScaled: false,
        previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
      },
    });
    const { change, configOf } = await renderChange(
      Schematic.Node.createConfig({ variant: "valve", stateChannel: 12 }),
    );
    act(() =>
      change(["n1"], { variant: "symbol", spec: { type: "custom", key: symbol.key } }),
    );
    await waitFor(async () =>
      expect(await configOf()).toMatchObject({
        variant: "custom_actuator",
        specKey: symbol.key,
        stateChannel: 12,
        label: { label: symbol.name },
      }),
    );
  });

  it("should rename a built-in symbol node that keeps the name it shows", async () => {
    const { change, configOf } = await renderChange(
      Schematic.Node.createConfig({
        ...builtIn("normally_open_solenoid_valve"),
        label: { label: "Normally open solenoid" },
      }),
    );
    act(() => change(["n1"], builtIn("valve")));
    await waitFor(async () =>
      expect(await configOf()).toMatchObject({
        ...builtIn("valve"),
        label: { label: "Valve" },
      }),
    );
  });

  it("should keep the bindings that the new symbol shares", async () => {
    const binding = { key: "active", source: { type: "local" as const, value: 1 } };
    const { change, configOf } = await renderChange(
      Schematic.Node.createConfig({
        ...builtIn("valve"),
        label: { label: "Valve" },
        states: [binding],
      }),
    );
    act(() => change(["n1"], builtIn("light")));
    await waitFor(async () =>
      expect(await configOf()).toMatchObject({
        ...builtIn("light"),
        label: { label: "Light" },
        states: [binding],
      }),
    );
  });

  it("should throw when the new symbol is missing", async () => {
    const { change } = await renderChange(
      Schematic.Node.createConfig(builtIn("valve")),
    );
    const key = uuid.create();
    expect(() =>
      change(["n1"], { variant: "symbol", spec: { type: "missing", key } }),
    ).toThrow(new NotFoundError(`symbol ${key} is missing`));
  });
});
