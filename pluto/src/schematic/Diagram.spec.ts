// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { edgeChangesToActions, nodeChangesToActions } from "@/schematic/Diagram";
import { Edge } from "@/schematic/edge";
import { type Diagram as Base } from "@/vis/diagram";

describe("nodeChangesToActions", () => {
  it("returns an empty array for empty input", () => {
    expect(nodeChangesToActions([])).toEqual([]);
  });

  it("translates a position change to a setNodePosition action", () => {
    const change: Base.NodeChange = {
      type: "position",
      key: "n1",
      position: { x: 10, y: 20 },
      dragging: true,
    };
    expect(nodeChangesToActions([change])).toEqual([
      schematic.setNodePosition({ key: "n1", position: { x: 10, y: 20 } }),
    ]);
  });

  it("translates a remove change to a removeNode action", () => {
    const change: Base.NodeChange = { type: "remove", key: "n1" };
    expect(nodeChangesToActions([change])).toEqual([
      schematic.removeNode({ key: "n1" }),
    ]);
  });

  it("silently drops select changes since selection lives outside the schematic", () => {
    const changes: Base.NodeChange[] = [
      { type: "select", key: "n1", selected: true },
      { type: "select", key: "n2", selected: false },
    ];
    expect(nodeChangesToActions(changes)).toEqual([]);
  });

  it("preserves input order across a mixed batch of change types", () => {
    const changes: Base.NodeChange[] = [
      { type: "position", key: "n1", position: { x: 1, y: 2 }, dragging: false },
      { type: "select", key: "n3", selected: true },
      { type: "remove", key: "n2" },
    ];
    expect(nodeChangesToActions(changes)).toEqual([
      schematic.setNodePosition({ key: "n1", position: { x: 1, y: 2 } }),
      schematic.removeNode({ key: "n2" }),
    ]);
  });
});

describe("edgeChangesToActions", () => {
  const newEdge = (key: string): schematic.Edge => ({
    key,
    source: { node: "n1", param: "out" },
    target: { node: "n2", param: "in" },
  });

  it("returns an empty array for empty input", () => {
    expect(edgeChangesToActions([])).toEqual([]);
  });

  it("translates an add change into addEdge followed by setConfig", () => {
    const edge = newEdge("e1");
    const out = edgeChangesToActions([{ type: "add", edge }]);
    expect(out).toHaveLength(2);
    expect(out[0]).toEqual(schematic.addEdge({ edge }));
    expect(out[1].type).toBe("set_config");
    expect(out[1].type === "set_config" && out[1].setConfig.key).toBe("e1");
  });

  it("initializes the new edge's config with the pipe variant's default", () => {
    const edge = newEdge("e1");
    const [, setCfg] = edgeChangesToActions([{ type: "add", edge }]);
    const expected = Edge.REGISTRY.pipe.defaultConfig() as unknown as record.Unknown;
    expect(setCfg).toEqual(schematic.setConfig({ key: "e1", config: expected }));
  });

  it("translates a remove change to a removeEdge action", () => {
    expect(edgeChangesToActions([{ type: "remove", key: "e1" }])).toEqual([
      schematic.removeEdge({ key: "e1" }),
    ]);
  });

  it("silently drops select changes since selection lives outside the schematic", () => {
    const changes: Base.EdgeChange[] = [
      { type: "select", key: "e1", selected: true },
      { type: "select", key: "e2", selected: false },
    ];
    expect(edgeChangesToActions(changes)).toEqual([]);
  });

  it("preserves input order and emits a separate pair for each add", () => {
    const e1 = newEdge("e1");
    const e3 = newEdge("e3");
    const changes: Base.EdgeChange[] = [
      { type: "add", edge: e1 },
      { type: "select", key: "e2", selected: true },
      { type: "remove", key: "e2" },
      { type: "add", edge: e3 },
    ];
    const pipeDefault = Edge.REGISTRY.pipe.defaultConfig() as unknown as record.Unknown;
    expect(edgeChangesToActions(changes)).toEqual([
      schematic.addEdge({ edge: e1 }),
      schematic.setConfig({ key: "e1", config: pipeDefault }),
      schematic.removeEdge({ key: "e2" }),
      schematic.addEdge({ edge: e3 }),
      schematic.setConfig({ key: "e3", config: pipeDefault }),
    ]);
  });
});
