import { schematic } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";

import { Component } from "@/component";
import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";
import { Diagram as Base } from "@/vis/diagram";

export const Diagram = Base.create({
  node: Component.renderProp(Node.Renderer),
  edge: Component.renderProp(Edge.Renderer),
  connectionLine: Component.renderProp(Edge.ConnectionLine),
});

export const nodeChangesToActions = (
  changes: Base.NodeChange[],
): schematic.Action[] => {
  const actions: schematic.Action[] = [];
  changes.forEach((ch) => {
    switch (ch.type) {
      case "position":
        actions.push(schematic.setNodePosition({ key: ch.key, position: ch.position }));
        return;
      case "dimensions":
        actions.push(
          schematic.setNodeMeasured({ key: ch.key, measured: ch.dimensions }),
        );
        return;
      case "remove":
        actions.push(schematic.removeNode({ key: ch.key }));
    }
  });
  return actions;
};

export const edgeChangesToActions = (changes: Base.EdgeChange[]): schematic.Action[] =>
  changes.flatMap((ch) => {
    switch (ch.type) {
      case "add":
        return [
          schematic.addEdge({ edge: ch.edge }),
          schematic.setConfig({
            key: ch.edge.key,
            config: Edge.REGISTRY.pipe.defaultConfig() as unknown as record.Unknown,
          }),
        ];
      case "remove":
        return [schematic.removeEdge({ key: ch.key })];
      default:
        return [];
    }
  });
