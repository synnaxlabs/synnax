// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource, schematic } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Change } from "@/feature/vcs/diff/change";

const NAMES = new Map([
  ["channel:1048577", "valve_state"],
  ["channel:1048578", "valve_state_2"],
]);
const name: Change.Name = (id) => NAMES.get(resource.idToString(id));

const symbolName = ({ variant }: schematic.ElementConfig): string =>
  Change.humanize(variant);

const createSchematic = (doc: Partial<schematic.New> = {}): schematic.Schematic =>
  schematic.schematicZ.parse({
    name: "Engine",
    nodes: [
      { key: "fuel", position: { x: 0, y: 0 } },
      { key: "pump", position: { x: 200, y: 0 } },
    ],
    edges: [
      {
        key: "line",
        source: { node: "fuel", param: "outlet" },
        target: { node: "pump", param: "inlet" },
      },
    ],
    configs: {
      fuel: { variant: "valve", label: { label: "Fuel valve" }, stateChannel: 1048577 },
      pump: { variant: "pump", label: { label: "Fuel pump" } },
      line: { variant: "pipe" },
    },
    ...doc,
  });

const BEFORE = createSchematic();

const diff = (after: schematic.Schematic, before = BEFORE): Change.Elements =>
  Change.diffElements(before, after, { name, symbolName });

describe("Change", () => {
  describe("diffElements", () => {
    it("should report nothing for equal schematics", () => {
      const result = diff(createSchematic());
      expect(result.elements).toEqual([]);
      expect(result.document).toEqual([]);
      expect(result.before.size).toBe(0);
      expect(result.after.size).toBe(0);
    });

    it("should report an added symbol by its label", () => {
      const result = diff(
        createSchematic({
          nodes: [...BEFORE.nodes, { key: "vent", position: { x: 0, y: 200 } }],
          configs: {
            ...BEFORE.configs,
            vent: { variant: "vent", label: { label: "" } },
          },
        }),
      );
      expect(result.elements).toEqual([
        {
          key: "vent",
          variant: "symbol",
          label: "Vent",
          kind: "added",
          properties: [],
        },
      ]);
      expect([...result.after]).toEqual([["vent", "added"]]);
      expect(result.before.size).toBe(0);
    });

    it("should report a removed symbol and the connection attached to it", () => {
      const result = diff(
        createSchematic({
          nodes: [BEFORE.nodes[0]],
          edges: [],
          configs: { fuel: BEFORE.configs.fuel },
        }),
      );
      expect(result.elements).toEqual([
        {
          key: "pump",
          variant: "symbol",
          label: "Fuel pump",
          kind: "removed",
          properties: [],
        },
        {
          key: "line",
          variant: "connection",
          label: "Pipe from Fuel valve to Fuel pump",
          kind: "removed",
          properties: [],
        },
      ]);
      expect([...result.before]).toEqual([["pump", "removed"]]);
      expect(result.after.size).toBe(0);
    });

    it("should report each changed property of a symbol", () => {
      const result = diff(
        createSchematic({
          configs: {
            ...BEFORE.configs,
            fuel: {
              variant: "valve",
              label: { label: "Main fuel valve" },
              stateChannel: 1048578,
            },
          },
        }),
      );
      expect(result.elements).toHaveLength(1);
      const [element] = result.elements;
      expect(element).toMatchObject({
        key: "fuel",
        variant: "symbol",
        label: "Main fuel valve",
        kind: "changed",
      });
      expect(element.properties.map(Change.format)).toEqual([
        "Label: Fuel valve -> Main fuel valve",
        "State channel: valve_state -> valve_state_2",
      ]);
      expect(element.properties[1].after).toEqual([
        { text: "valve_state_2", resource: { type: "channel", key: "1048578" } },
      ]);
      expect([...result.before]).toEqual([["fuel", "changed"]]);
      expect([...result.after]).toEqual([["fuel", "changed"]]);
    });

    it("should report a moved symbol with its rounded position", () => {
      const result = diff(
        createSchematic({
          nodes: [{ key: "fuel", position: { x: 40.4, y: 19.6 } }, BEFORE.nodes[1]],
        }),
      );
      expect(result.elements).toEqual([
        {
          key: "fuel",
          variant: "symbol",
          label: "Fuel valve",
          kind: "changed",
          properties: [
            {
              path: ["Position"],
              kind: "changed",
              before: [{ text: "0, 0" }],
              after: [{ text: "40, 20" }],
            },
          ],
        },
      ]);
    });

    it("should name a symbol without a label by its kind", () => {
      const result = diff(
        createSchematic({
          configs: {
            ...BEFORE.configs,
            pump: { variant: "pump", label: { label: " " } },
          },
        }),
      );
      expect(result.elements.map(({ label }) => label)).toEqual(["Pump"]);
    });

    it("should name a symbol without a config as a symbol", () => {
      const result = diff(
        createSchematic({
          nodes: [...BEFORE.nodes, { key: "bare", position: { x: 0, y: 0 } }],
        }),
      );
      expect(result.elements.map(({ label }) => label)).toEqual(["Symbol"]);
    });

    it("should report an added connection by its kind and ends", () => {
      const result = diff(
        createSchematic({
          edges: [
            ...BEFORE.edges,
            {
              key: "return",
              source: { node: "pump", param: "outlet" },
              target: { node: "fuel", param: "inlet" },
            },
          ],
        }),
      );
      expect(result.elements).toEqual([
        {
          key: "return",
          variant: "connection",
          label: "Connection from Fuel pump to Fuel valve",
          kind: "added",
          properties: [],
        },
      ]);
      expect(result.after.size).toBe(0);
    });

    it("should report a changed connection without adding it to the kind maps", () => {
      const result = diff(
        createSchematic({
          edges: [
            {
              ...BEFORE.edges[0],
              target: { node: "pump", param: "bypass" },
            },
          ],
          configs: { ...BEFORE.configs, line: { variant: "electric" } },
        }),
      );
      expect(result.elements).toHaveLength(1);
      const [element] = result.elements;
      expect(element).toMatchObject({
        key: "line",
        variant: "connection",
        label: "Electric from Fuel valve to Fuel pump",
        kind: "changed",
      });
      expect(element.properties.map(Change.format)).toEqual([
        "Target › Param: inlet -> bypass",
        "Variant: pipe -> electric",
      ]);
      expect(result.before.size).toBe(0);
      expect(result.after.size).toBe(0);
    });

    it("should not report a connection whose route alone changed", () => {
      const result = diff(
        createSchematic({
          configs: {
            ...BEFORE.configs,
            line: {
              variant: "pipe",
              segments: [{ direction: "x", length: 120 }],
            },
          },
        }),
      );
      expect(result.elements).toEqual([]);
    });

    it("should report a change to the schematic itself as a document change", () => {
      const result = diff(createSchematic({ name: "Engine v2" }));
      expect(result.elements).toEqual([]);
      expect(result.document.map(Change.format)).toEqual(["Name: Engine -> Engine v2"]);
    });

    it("should list added and changed symbols before removed ones", () => {
      const result = diff(
        createSchematic({
          nodes: [
            { key: "vent", position: { x: 0, y: 200 } },
            { key: "fuel", position: { x: 10, y: 0 } },
          ],
          edges: [],
          configs: {
            vent: { variant: "vent", label: { label: "Vent" } },
            fuel: BEFORE.configs.fuel,
          },
        }),
      );
      expect(result.elements.map(({ key, kind }) => [key, kind])).toEqual([
        ["vent", "added"],
        ["fuel", "changed"],
        ["pump", "removed"],
        ["line", "removed"],
      ]);
      expect(Object.fromEntries(result.before)).toEqual({
        fuel: "changed",
        pump: "removed",
      });
      expect(Object.fromEntries(result.after)).toEqual({
        vent: "added",
        fuel: "changed",
      });
    });
  });
});
