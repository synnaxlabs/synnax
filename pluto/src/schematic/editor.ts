// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  NotFoundError,
  query,
  schematic,
  type Synnax as Client,
} from "@synnaxlabs/client";
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
      const sym = client?.schematics.symbols.watch(config.specKey).getCached();
      if (query.isLive(sym)) config.label.label = sym.name;
    }
    dispatch(
      schematic.setNode({ node: { key, position: position ?? xy.ZERO, config } }),
    );
  };
};

// The label a symbol takes on creation: the name of its custom or built-in symbol.
const defaultLabel = (client: Client, config: Node.Config): string | undefined => {
  const key = Node.customKey(config);
  if (key != null) {
    const sym = client.schematics.symbols.watch(key).getCached();
    return query.isLive(sym) ? sym.name : undefined;
  }
  if (config.variant === "symbol")
    return Node.BUILT_IN_NAMES.get(Node.symbolKey(config));
  const spec = Node.resolveSpec(config.variant);
  return spec.label ?? spec.name;
};

// The spec that a symbol node draws, or undefined when its symbol is missing, as the
// node draws it.
const resolveSymbol = (
  client: Client,
  ref: schematic.symbol.Ref,
): schematic.symbol.Spec | undefined => {
  if (ref.type === "builtin") return Node.BUILT_INS.get(ref.name);
  if (ref.type === "missing") return undefined;
  const sym = client.schematics.symbols.watch(ref.key).getCached();
  return query.isLive(sym) ? sym.data : undefined;
};

// Changes an old node to the old node of the symbol that the reference names, so it
// keeps its channels.
const changeLegacy = (
  client: Client,
  config: Node.Config,
  ref: schematic.symbol.Ref,
): Node.Config => {
  if (ref.type === "builtin")
    return Node.changeConfig(config, { variant: Node.variantZ.parse(ref.name) });
  const sym =
    ref.type === "custom"
      ? client.schematics.symbols.watch(ref.key).getCached()
      : undefined;
  if (!query.isLive(sym)) throw new NotFoundError(`symbol ${ref.key} is missing`);
  return Node.changeConfig(config, {
    variant: Node.customVariant(sym),
    specKey: ref.key,
  });
};

/**
 * useChangeSymbol returns a callback that changes the given symbols to the symbol the
 * input names, dispatched as a single undoable step. A label that still reads the old
 * symbol's name takes the new symbol's name. A symbol node keeps the bindings that
 * {@link Node.rebind} keeps. Any other node changes to the old node of the symbol, so
 * it keeps its channels.
 * @throws {NotFoundError} if the input names a symbol that is missing.
 */
export const useChangeSymbol = () => {
  const key = Scope.use();
  const client = Synnax.use();
  const dispatch = useSingleDispatch();
  return <V extends Node.Variant>(keys: readonly string[], input: Node.Input<V>) => {
    const s = client?.schematics.watch({ key }).getCached();
    if (client == null || !query.isLive(s)) return;
    const actions = keys.flatMap((k) => {
      const config = s.nodes.find((n) => n.key === k)?.config;
      if (config == null || !Node.isConfig(config))
        throw new Error(`Element ${k} is not a symbol`);
      let next: Node.Config = Node.changeConfig(config, input);
      if (config.variant !== "symbol" && next.variant === "symbol")
        next = changeLegacy(client, config, next.spec);
      if (Node.symbolKey(next) === Node.symbolKey(config)) return [];
      if (config.variant === "symbol" && next.variant === "symbol") {
        const to = resolveSymbol(client, next.spec);
        if (to == null)
          throw new NotFoundError(`symbol ${Node.symbolKey(next)} is missing`);
        const bindings = Node.rebind(config, resolveSymbol(client, config.spec), to);
        next.states = bindings.states;
        next.commands = bindings.commands;
      }
      if (
        "label" in config &&
        "label" in next &&
        config.label.label === defaultLabel(client, config)
      )
        next.label.label = defaultLabel(client, next) ?? next.label.label;
      return schematic.setChanged({ key: k, base: config, next });
    });
    if (actions.length > 0) dispatch(actions);
  };
};
