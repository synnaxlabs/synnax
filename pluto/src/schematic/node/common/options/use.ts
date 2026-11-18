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
 * Returns the states that a node draws: the entries of its enumeration, or its options
 * when it names none. Returns no states while the enumeration loads.
 */
export const use = ({
  enumeration: key,
  options,
}: UseParams): schematic.StateMapping[] => {
  const resolve = Color.useResolve();
  const { data } = Enumeration.useResultMultiple({ keys: key == null ? [] : [key] });
  if (key == null) return options;
  return (data?.[0]?.entries ?? []).map(({ color, ...entry }) => ({
    key: entry.key,
    name: entry.name,
    value: entry.value,
    color: color == null ? undefined : resolve(color),
  }));
};
