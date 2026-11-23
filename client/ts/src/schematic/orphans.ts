// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Command, State } from "@/schematic/symbol/types.gen";
import type { SymbolNodeConfig } from "@/schematic/types.gen";

/** A field of a state binding that replaces a default of its state. */
export type Override = "bounds" | "enumeration" | "match" | "default";

const LACKED_OVERRIDES: Record<State["type"], Override[]> = {
  number: ["enumeration", "match", "default"],
  enumeration: ["bounds"],
};

/** The bindings and overrides of a symbol node that a symbol orphans. */
export interface Orphans {
  /** The keys of the state bindings whose state the symbol lacks. */
  states: string[];
  /** The overrides that the type of their state lacks, by state binding key. */
  overrides: { key: string; fields: Override[] }[];
  /** The keys of the bindings of commands that are missing or that have a state. */
  commands: string[];
}

/**
 * @returns the bindings and overrides of config that a symbol with states and commands
 * orphans. A save of the symbol removes them from each node that draws it.
 */
export const orphans = (
  config: Pick<SymbolNodeConfig, "states" | "commands">,
  states: State[],
  commands: Command[],
): Orphans => {
  const result: Orphans = { states: [], overrides: [], commands: [] };
  for (const b of config.states) {
    const st = states.find((s) => s.key === b.key);
    if (st == null) {
      result.states.push(b.key);
      continue;
    }
    const fields = LACKED_OVERRIDES[st.type].filter((f) => b[f] != null);
    if (fields.length > 0) result.overrides.push({ key: b.key, fields });
  }
  for (const b of config.commands) {
    const cmd = commands.find((c) => c.key === b.key);
    if (cmd == null || cmd.state != null) result.commands.push(b.key);
  }
  return result;
};
