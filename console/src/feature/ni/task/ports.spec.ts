// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  analogInputClaims,
  analogInputPorts,
  groups,
  lowestFreePort,
} from "@/feature/ni/task/ports";
import { createHardware } from "@/feature/ni/task/testutil";

const hardware = createHardware();

const aiChannel = (
  key: string,
  port: number,
  overrides: Partial<NI.Task.AIChannel> = {},
): NI.Task.AIChannel =>
  ({
    ...NI.Task.createAIChannel(),
    key,
    port,
    device: "dev",
    ...overrides,
  }) as NI.Task.AIChannel;

const layout = (channels: NI.Task.AIChannel[], selected: string[] = []) =>
  groups({
    ports: analogInputPorts(hardware),
    claims: analogInputClaims(channels, "dev", hardware),
    prefix: "ai",
    selected,
    describe: (key) => `chan_${key}`,
  });

const cellOf = (channels: NI.Task.AIChannel[], name: string, selected?: string[]) => {
  const cell = layout(channels, selected)
    .flatMap(({ cells }) => cells)
    .find((c) => c.name === name);
  if (cell == null) throw new Error(`no cell named ${name}`);
  return cell;
};

describe("groups", () => {
  it("should put each negative input under its port, eight pairs to a group", () => {
    const [first, second] = layout([]);
    expect(first.columns).toBe(8);
    expect(first.cells.map(({ name, row, column }) => [name, row, column])).toEqual(
      Array.from({ length: 8 }, (_, i) => [
        [`ai${i}`, 1, i + 1],
        [`ai${i + 8}`, 2, i + 1],
      ]).flat(),
    );
    expect(second).toBeUndefined();
  });

  it("should span a differential channel across both ports of its pair", () => {
    const cell = cellOf([aiChannel("a", 2)], "ai2", ["a"]);
    expect(cell).toMatchObject({
      labels: ["2", "10"],
      paired: true,
      state: "used",
      selected: true,
      channel: "a",
      tooltip: "ai2 and ai10: chan_a",
    });
  });

  it("should keep the pair apart for a channel that is not differential", () => {
    const channels = [aiChannel("a", 2, { terminalConfig: "RSE" })];
    expect(cellOf(channels, "ai2")).toMatchObject({ paired: false, state: "used" });
    expect(cellOf(channels, "ai10")).toMatchObject({
      state: "free",
      tooltip: "ai10 is free",
    });
  });

  it("should mark a differential pair as a conflict when another channel holds a port", () => {
    const channels = [aiChannel("a", 8), aiChannel("b", 0)];
    expect(cellOf(channels, "ai0")).toMatchObject({
      paired: true,
      state: "conflict",
      channel: "b",
      tooltip: "ai0 and ai8 are used by chan_b and chan_a",
    });
  });

  it("should mark a port two channels hold as a conflict", () => {
    const channels = [
      aiChannel("a", 9, { terminalConfig: "RSE" }),
      aiChannel("b", 9, { terminalConfig: "RSE" }),
    ];
    expect(cellOf(channels, "ai9")).toMatchObject({
      state: "conflict",
      tooltip: "ai9 is used by chan_a and chan_b",
    });
  });
});

describe("lowestFreePort", () => {
  it("should skip a port whose negative input another channel holds", () => {
    const channels = [aiChannel("a", 9, { terminalConfig: "RSE" })];
    const claims = analogInputClaims(channels, "dev", hardware);
    const holds = ({ port }: { port: number }) =>
      port < 8 ? [port, port + 8] : [port];
    expect(lowestFreePort(analogInputPorts(hardware), claims, holds)).toBe(0);
    expect(
      lowestFreePort(
        analogInputPorts(hardware).filter(({ port }) => port === 1),
        claims,
        holds,
      ),
    ).toBe(2);
  });

  it("should count up from 0 without the device's ports", () => {
    expect(
      lowestFreePort(
        null,
        new Map([
          [0, []],
          [1, []],
        ]),
      ),
    ).toBe(2);
  });
});
