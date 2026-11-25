// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration } from "@synnaxlabs/client";
import type { color } from "@synnaxlabs/x";

import { Color } from "@/color";
import { Enumeration } from "@/enumeration";

/** A state that a node draws: an entry of its enumeration, with its color resolved. */
export interface State extends Pick<enumeration.Entry, "key" | "name" | "value"> {
  color?: color.Color;
}

/**
 * Returns the states that a node draws: the entries of the enumeration that it names.
 * Returns no states for a node that names none, and while the enumeration loads.
 */
export const use = (key: enumeration.Key | undefined): State[] => {
  const resolve = Color.useResolve();
  const { data } = Enumeration.useResult(key == null ? null : { key });
  return (data?.entries ?? []).map(({ key, name, value, color }) => ({
    key,
    name,
    value,
    color: color == null ? undefined : resolve(color),
  }));
};
