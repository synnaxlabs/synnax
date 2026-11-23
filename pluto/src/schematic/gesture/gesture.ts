// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";

import { Edge } from "@/schematic/edge";
import type { Diagram as Base } from "@/vis/diagram";

export const nodeChangesToActions = (
  changes: Base.NodeChange[],
): schematic.Action[] => {
  const actions: schematic.Action[] = [];
  changes.forEach((ch) => {
    switch (ch.type) {
      case "position":
        actions.push(schematic.setNodePosition({ key: ch.key, position: ch.position }));
        return;
      case "remove":
        actions.push(schematic.removeNode({ key: ch.key }));
        break;
      case "select":
        break;
    }
  });
  return actions;
};

export const edgeChangesToActions = (changes: Base.EdgeChange[]): schematic.Action[] =>
  changes.flatMap((ch) => {
    switch (ch.type) {
      case "add":
        return [
          schematic.addEdge({
            edge: { ...ch.edge, config: Edge.REGISTRY.pipe.defaultConfig() },
          }),
        ];
      case "remove":
        return [schematic.removeEdge({ key: ch.key })];
      default:
        return [];
    }
  });
