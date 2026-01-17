// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Diagram } from "@synnaxlabs/pluto";
import { box, id } from "@synnaxlabs/x";

import { Layout } from "@/layout";
import { moveMosaicTab } from "@/layout/slice";
import { registerMacro } from "@/perf/macros/registry";
import { type MacroContext, type MacroStep } from "@/perf/macros/types";
import { Schematic } from "@/schematic";
import { selectRequired } from "@/schematic/selectors";
import { addElement, selectAll, setNodePositions } from "@/schematic/slice";

const ESTIMATED_NODE_SIZE = { width: 60, height: 60 };

const SYMBOLS = ["valve", "pump", "tank", "light", "switch", "button"] as const;

export const schematicMacro: MacroStep[] = [
  {
    name: "Create Schematic",
    execute: async (ctx: MacroContext) => {
      const { key } = ctx.placer(
        Schematic.create({
          name: `Perf Test Schematic ${Date.now()}`,
          location: "mosaic",
        }),
      );
      ctx.createdLayoutKeys.push(key);
    },
  },
  {
    name: "Snap to Right",
    execute: async (ctx: MacroContext) => {
      const schematicKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (schematicKey == null) return;
      ctx.dispatch(
        moveMosaicTab({
          windowKey: MAIN_WINDOW,
          key: 1,
          tabKey: schematicKey,
          loc: "right",
        }),
      );
    },
  },
  {
    name: "Add Symbols",
    execute: async (ctx: MacroContext) => {
      const schematicKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (schematicKey == null) return;

      SYMBOLS.forEach((symbol, i) => {
        ctx.dispatch(
          addElement({
            key: schematicKey,
            elKey: id.create(),
            props: { key: symbol },
            node: {
              position: { x: 100 + (i % 3) * 150, y: 100 + Math.floor(i / 3) * 150 },
            },
          }),
        );
      });
    },
  },
  {
    name: "Select All Symbols",
    execute: async (ctx: MacroContext) => {
      const schematicKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (schematicKey == null) return;
      ctx.dispatch(selectAll({ key: schematicKey }));
    },
  },
  {
    name: "Align Center & Distribute",
    execute: async (ctx: MacroContext) => {
      const schematicKey = ctx.createdLayoutKeys[ctx.createdLayoutKeys.length - 1];
      if (schematicKey == null) return;

      const state = ctx.store.getState();
      const schematic = selectRequired(state, schematicKey);
      const nodes = schematic.nodes;

      if (nodes.length === 0) return;

      let layouts = nodes.map(
        (node) =>
          new Diagram.NodeLayout(
            node.key,
            box.construct(node.position, ESTIMATED_NODE_SIZE),
            [],
          ),
      );
      layouts = Diagram.alignNodesAlongDirection(layouts, "x");
      layouts = Diagram.distributeNodes(layouts, "x");
      ctx.dispatch(
        setNodePositions({
          key: schematicKey,
          positions: layouts.map((l) => [l.key, box.topLeft(l.box)]),
        }),
      );
    },
  },
  {
    name: "Close Schematic",
    execute: async (ctx: MacroContext) => {
      const schematicKey = ctx.createdLayoutKeys.pop();
      if (schematicKey == null) return;
      ctx.dispatch(Layout.remove({ keys: [schematicKey] }));
    },
  },
];

registerMacro({
  type: "schematic",
  name: "Schematic",
  description:
    "Creates a schematic, snaps to right, adds symbols, aligns and distributes, then closes",
  category: "schematic",
  factory: () => schematicMacro,
});
