// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { strings } from "@synnaxlabs/x";

type Named = Pick<schematic.symbol.Spec, "states" | "commands">;

const nameOf = (items: { key: string; name: string }[], key: string): string =>
  items.find((i) => i.key === key)?.name ?? key;

/**
 * @returns what a node loses with its orphans, such as "the Level binding". A binding
 * takes its name from before, the spec that it binds, and an override from after.
 */
export const losses = (
  { states, overrides, commands }: schematic.Orphans,
  before: Named,
  after: Named,
): string[] => [
  ...states.map((k) => `the ${nameOf(before.states, k)} binding`),
  ...overrides.map(
    ({ key, fields }) =>
      `the ${strings.naturalLanguageJoin(fields)} of ${nameOf(after.states, key)}`,
  ),
  ...commands.map((k) => `the ${nameOf(before.commands, k)} binding`),
];
