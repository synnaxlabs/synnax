// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { json } from "@synnaxlabs/x";

/**
 * @returns a name for the field at each pointer, apart from taken and from each other,
 * also as a channel name. The tokens of a pointer join with "_", so "/motor/speed"
 * names "motor_speed". A pointer with no name, such as the whole document, names
 * "value". A taken name gets a number, such as "value_2".
 */
export const fieldNames = (pointers: string[], taken: string[] = []): string[] => {
  const names = new Set(taken.map((name) => channel.escapeInvalidName(name)));
  return pointers.map((pointer) => {
    const base = json.parsePointer(pointer).join("_") || "value";
    let name = base;
    for (let n = 2; names.has(channel.escapeInvalidName(name)); n++)
      name = `${base}_${n}`;
    names.add(channel.escapeInvalidName(name));
    return name;
  });
};
