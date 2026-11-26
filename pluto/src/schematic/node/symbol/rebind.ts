// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";

type Bindings = Pick<schematic.SymbolNodeConfig, "states" | "commands">;

/**
 * @returns the bindings of a symbol node whose spec changes from `from` to `to`. A
 * state binding stays whole when `to` has a state of its key and type, and keeps only
 * its channel source when the type differs or `from` is missing. A command binding
 * stays when `to` has its command and that command writes no state. Every other
 * binding is dropped.
 */
export const rebind = (
  bindings: Bindings,
  from: schematic.symbol.Spec | undefined,
  to: schematic.symbol.Spec,
): Bindings => {
  const { states, commands } = bindings;
  const fromTypes = new Map(from?.states.map((s) => [s.key, s.type]));
  const toTypes = new Map(to.states.map((s) => [s.key, s.type]));
  const orphaned = new Set(
    schematic.orphans(bindings, to.states, to.commands).commands,
  );
  return {
    states: states.flatMap((binding) => {
      const type = toTypes.get(binding.key);
      if (type == null) return [];
      if (type === fromTypes.get(binding.key)) return [binding];
      const { key, source } = binding;
      return source?.type === "channel" ? [{ key, source }] : [];
    }),
    commands: commands.filter(({ key }) => !orphaned.has(key)),
  };
};
