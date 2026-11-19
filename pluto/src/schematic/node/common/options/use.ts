// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, type schematic } from "@synnaxlabs/client";

import { Color } from "@/color";
import { Enumeration } from "@/enumeration";

export interface UseParams {
  /** The enumeration that the node names, if any. */
  enumeration?: enumeration.Key;
  /** The states that the node draws when it names no enumeration. */
  options: schematic.StateMapping[];
}

/**
 * @param key - The enumeration that the node names, if any.
 * @returns whether the node draws its own options, which it does when it names no
 * enumeration.
 */
export const drawn = (key?: enumeration.Key): key is undefined => key == null;

/**
 * Returns the states that a node draws: the entries of its enumeration, or its options
 * when it names none. Returns no states while the enumeration loads.
 */
export const use = ({
  enumeration: key,
  options,
}: UseParams): schematic.StateMapping[] => {
  const resolve = Color.useResolve();
  const { data } = Enumeration.useResultMultiple({ keys: drawn(key) ? [] : [key] });
  if (drawn(key)) return options;
  return (data?.[0]?.entries ?? []).map(({ color, ...entry }) => ({
    key: entry.key,
    name: entry.name,
    value: entry.value,
    color: color == null ? undefined : resolve(color),
  }));
};
