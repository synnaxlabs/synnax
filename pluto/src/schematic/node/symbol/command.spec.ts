// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { BUILT_INS } from "@/schematic/node/builtIns";
import { Telem } from "@/schematic/node/common/telem";
import { bind } from "@/schematic/node/symbol/bind";
import {
  clickable,
  commandChannels,
  commandsOf,
  fixedValue,
  interactionAt,
  pick,
  sinks,
  stateOf,
  target,
} from "@/schematic/node/symbol/command";

const VALVE = BUILT_INS.get("valve")!;
const [DEACTIVATE, ACTIVATE] = VALVE.commands;
const [CLICK] = VALVE.interactions;

const ASK: schematic.symbol.Command = {
  key: "set",
  name: "Set",
  state: "active",
  value: { type: "ask" },
};
const PULSE: schematic.symbol.Command = {
  key: "pulse",
  name: "Pulse",
  value: { type: "fixed", value: 1 },
};

const createBindings = (
  bindings: Partial<Pick<schematic.SymbolNodeConfig, "states" | "commands">>,
): Pick<schematic.SymbolNodeConfig, "states" | "commands"> => ({
  states: [],
  commands: [],
  ...bindings,
});

const channelState = (command?: number): schematic.StateBinding => ({
  key: "active",
  source: { type: "channel", key: 1, command },
  units: "",
  notation: "standard",
});

const localState = (value: number): schematic.StateBinding => ({
  key: "active",
  source: { type: "local", value },
  units: "",
  notation: "standard",
});

describe("command", () => {
  describe("target", () => {
    it("should write to the command channel of the state's channel source", () => {
      expect(target(ACTIVATE, createBindings({ states: [channelState(2)] }))).toEqual({
        type: "channel",
        channel: 2,
      });
    });

    it("should write to the local value of the state", () => {
      expect(target(ACTIVATE, createBindings({ states: [localState(0)] }))).toEqual({
        type: "local",
        state: "active",
      });
    });

    it.each([
      ["a channel source with no command channel", [channelState()]],
      ["a state with no binding", []],
    ])("should have no target for %s", (_, states) => {
      expect(target(ACTIVATE, createBindings({ states }))).toBeUndefined();
    });

    it("should write a command with no state to its command binding", () => {
      const bindings = createBindings({ commands: [{ key: "pulse", channel: 3 }] });
      expect(target(PULSE, bindings)).toEqual({ type: "channel", channel: 3 });
      expect(target(PULSE, createBindings({}))).toBeUndefined();
    });
  });

  describe("commandChannels", () => {
    it("should list each channel that the commands write once", () => {
      const bindings = createBindings({ states: [channelState(2)] });
      expect(commandChannels(VALVE, bindings)).toEqual([2]);
    });

    it("should list no channel for a local state or no spec", () => {
      expect(
        commandChannels(VALVE, createBindings({ states: [localState(0)] })),
      ).toEqual([]);
      expect(commandChannels(undefined, createBindings({}))).toEqual([]);
    });
  });

  describe("sinks", () => {
    it("should build a number sink for each channel, by its key", () => {
      expect(sinks([2, 3])).toEqual({
        2: Telem.numberSink(2),
        3: Telem.numberSink(3),
      });
    });
  });

  describe("commandsOf", () => {
    it("should list the commands of an interaction in its order", () => {
      expect(commandsOf(VALVE, CLICK)).toEqual([DEACTIVATE, ACTIVATE]);
    });

    it("should throw for a command that the spec does not have", () => {
      const fire: schematic.symbol.Interaction = {
        type: "fire",
        key: "x",
        command: "x",
      };
      expect(() => commandsOf(VALVE, fire)).toThrow("symbol has no command x");
    });
  });

  describe("clickable", () => {
    it("should keep an interaction whose commands have a target", () => {
      expect(clickable(VALVE, createBindings({ states: [localState(0)] }))).toEqual([
        CLICK,
      ]);
    });

    it("should drop an interaction whose commands have no target", () => {
      expect(clickable(VALVE, createBindings({ states: [channelState()] }))).toEqual(
        [],
      );
    });

    it("should keep an interaction that sends a command that asks", () => {
      const fire: schematic.symbol.Interaction = {
        type: "fire",
        key: "set",
        command: "set",
      };
      const spec = { ...VALVE, commands: [ASK], interactions: [fire] };
      expect(clickable(spec, createBindings({ states: [localState(0)] }))).toEqual([
        fire,
      ]);
    });
  });

  describe("stateOf", () => {
    it("should return the state that a command writes", () => {
      expect(stateOf(VALVE, ACTIVATE)).toEqual(VALVE.states[0]);
    });

    it("should throw for a command with no state", () => {
      expect(() => stateOf(VALVE, PULSE)).toThrow("command pulse has no state");
    });
  });

  describe("fixedValue", () => {
    it("should return the value of a fixed command", () => {
      expect(fixedValue(ACTIVATE)).toBe(1);
    });

    it("should throw for a command that asks", () => {
      expect(() => fixedValue(ASK)).toThrow("command set asks");
    });
  });

  describe("pick", () => {
    const value = (v: number) => new Map([["active", v]]);
    const NONE = new Map<string, number>();

    it.each([
      ["open", 0, ACTIVATE],
      ["close", 1, DEACTIVATE],
    ])("should send %s when the state equals the other command", (_, v, command) => {
      expect(pick(VALVE, VALVE, CLICK, value(v), NONE)).toEqual(command);
    });

    it("should match the state value with the node's match", () => {
      const bound = bind.spec(VALVE, [
        { key: "active", match: { type: "within", bounds: { lower: 4, upper: 6 } } },
      ]);
      expect(pick(VALVE, bound, CLICK, value(5), NONE)).toEqual(DEACTIVATE);
      expect(pick(VALVE, bound, CLICK, value(1), NONE)).toEqual(ACTIVATE);
    });

    it("should send nothing when the command channel differs from the state", () => {
      expect(pick(VALVE, VALVE, CLICK, value(1), value(0))).toBeUndefined();
    });

    it("should send when the command channel agrees with the state", () => {
      expect(pick(VALVE, VALVE, CLICK, value(1), value(1))).toEqual(DEACTIVATE);
    });

    it("should send nothing when the state has no value", () => {
      expect(pick(VALVE, VALVE, CLICK, NONE, NONE)).toBeUndefined();
    });

    it("should send nothing when the value equals neither command", () => {
      const bound = bind.spec(VALVE, [{ key: "active", match: { type: "exact" } }]);
      expect(pick(VALVE, bound, CLICK, value(2), NONE)).toBeUndefined();
    });

    it("should send the command of a fire interaction", () => {
      const spec = { ...VALVE, commands: [PULSE] };
      const fire: schematic.symbol.Interaction = {
        type: "fire",
        key: "pulse",
        command: "pulse",
      };
      expect(pick(spec, spec, fire, NONE, NONE)).toEqual(PULSE);
    });
  });

  describe("interactionAt", () => {
    const root = document.createElement("div");
    root.innerHTML = `<svg><path class="handle"/><path class="body"/></svg>`;
    const [handle, body] = root.querySelectorAll("path");
    const PART: schematic.symbol.Interaction = {
      type: "fire",
      key: "part",
      command: "pulse",
      part: "handle",
    };
    const spec: schematic.symbol.Spec = {
      ...VALVE,
      parts: [{ key: "handle", name: "Handle", selectors: [".handle"] }],
    };

    it("should take the interaction of the part that holds the element", () => {
      expect(interactionAt(spec, [CLICK, PART], root, handle)).toEqual(PART);
    });

    it("should take the interaction on the whole drawing elsewhere", () => {
      expect(interactionAt(spec, [CLICK, PART], root, body)).toEqual(CLICK);
    });

    it("should take nothing outside the parts with no whole drawing click", () => {
      expect(interactionAt(spec, [PART], root, body)).toBeUndefined();
    });
  });
});
