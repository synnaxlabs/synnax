// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Renders an object path as a human-readable string.
 *
 * - An empty path becomes `<root>`.
 * - String segments are joined with `.`.
 * - Numeric segments are rendered as bracket indices (`items[0]`).
 *
 * @example
 * path([]) // "<root>"
 * path(["config", "channels", 0, "port"]) // "config.channels[0].port"
 */
export const path = (segments: ReadonlyArray<PropertyKey>): string => {
  if (segments.length === 0) return "<root>";
  let out = "";
  segments.forEach((p, i) => {
    if (typeof p === "number") out += `[${p}]`;
    else if (i === 0) out += String(p);
    else out += `.${String(p)}`;
  });
  return out;
};
