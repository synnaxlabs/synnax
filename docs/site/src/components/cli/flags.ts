// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type FlagType = "string" | "bool" | "int" | "duration" | "stringSlice";

export interface FlagDefinition {
  name: string;
  short?: string;
  type: FlagType;
  default: string | boolean | number | string[];
  description: string;
  persistent?: boolean;
}

export interface FlagRow {
  option: string;
  default: string;
  description: string;
}

export interface EnvRow {
  option: string;
  env: string;
}

const option = (def: FlagDefinition): string =>
  `--${def.name}${def.short != null && def.short.length > 0 ? `/-${def.short}` : ""}`;

const formatDefault = (def: FlagDefinition): string => {
  switch (def.type) {
    case "string":
    case "duration":
      return `"${def.default as string}"`;
    case "stringSlice":
      return "[]";
    case "bool":
    case "int":
      return String(def.default);
  }
};

const envName = (def: FlagDefinition): string =>
  `SYNNAX_${def.name.replace(/-/g, "_").toUpperCase()}`;

export const flagRows = (flags: readonly FlagDefinition[]): FlagRow[] =>
  flags.map((f) => ({
    option: option(f),
    default: formatDefault(f),
    description: f.description,
  }));

export const envRows = (flags: readonly FlagDefinition[]): EnvRow[] =>
  flags.map((f) => ({ option: option(f), env: envName(f) }));

export const filterFlags = (
  flags: readonly FlagDefinition[],
  names: readonly string[],
): FlagDefinition[] => {
  const set = new Set(names);
  return flags.filter((f) => set.has(f.name));
};
