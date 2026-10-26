// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, schematic, type Synnax as Client } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";

import { Node } from "@/schematic/node";
import { useSingleDispatch } from "@/schematic/queries";
import { Scope } from "@/schematic/scope";
import { Synnax } from "@/synnax";

// These hooks read the registry of every symbol, so they stay out of queries.ts, which
// the Console loads at startup.

export interface AddNodeProps<V extends Node.Variant = Node.Variant> {
  key: string;
  position?: xy.XY;
  config: Node.Input<V>;
}

export const useAddNode = () => {
  const client = Synnax.use();
  const dispatch = useSingleDispatch();

  return <V extends Node.Variant>({
    key,
    position,
    config: input,
  }: AddNodeProps<V>) => {
    const config: Node.Config = Node.createConfig(input);
    if (Node.isCustomConfig(config)) {
      const sym = client?.schematics.symbols.getCached(config.specKey);
      if (query.isLive(sym)) config.label.label = sym.name;
    }
    dispatch(
      schematic.setNode({ node: { key, position: position ?? xy.ZERO }, config }),
    );
  };
};

// The label a symbol takes on creation: a custom symbol's name, or its spec's label.
const defaultLabel = (client: Client, config: Node.Config): string | undefined => {
  if (!Node.isCustomConfig(config)) {
    const spec = Node.resolveSpec(config.variant);
    return spec.label ?? spec.name;
  }
  const sym = client.schematics.symbols.getCached(config.specKey);
  return query.isLive(sym) ? sym.name : undefined;
};

/**
 * useChangeSymbol returns a callback that changes the given symbols to the symbol the
 * input names, dispatched as a single undoable step. A label that still reads the old
 * symbol's name takes the new symbol's name.
 */
export const useChangeSymbol = () => {
  const key = Scope.use();
  const client = Synnax.use();
  const dispatch = useSingleDispatch();
  return <V extends Node.Variant>(keys: readonly string[], input: Node.Input<V>) => {
    const s = client?.schematics.getCached({ key });
    if (client == null || !query.isLive(s)) return;
    const actions = keys.flatMap((k) => {
      const config = s.configs[k];
      if (!Node.isConfig(config)) throw new Error(`Element ${k} is not a symbol`);
      const next = Node.changeConfig(config, input);
      if (Node.symbolKey(next) === Node.symbolKey(config)) return [];
      if (
        "label" in config &&
        "label" in next &&
        config.label.label === defaultLabel(client, config)
      )
        next.label.label = defaultLabel(client, next) ?? next.label.label;
      return [schematic.setConfig({ key: k, config: next })];
    });
    if (actions.length > 0) dispatch(actions);
  };
};
