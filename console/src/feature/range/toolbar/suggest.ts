// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, type Synnax } from "@synnaxlabs/client";

const byStart = (a: ranger.Range, b: ranger.Range): number =>
  a.timeRange.start.nanoseconds - b.timeRange.start.nanoseconds;

/** Adds one to a trailing number and keeps its width: "HF-302" gives "HF-303". */
const increment = (name: string): string | null => {
  const match = /^(.*?)(\d+)$/.exec(name);
  if (match == null) return null;
  const [, head, digits] = match;
  return `${head}${String(Number(digits) + 1).padStart(digits.length, "0")}`;
};

export interface Suggestion {
  name: string;
  /** The color of the range that the name came from. */
  color?: ranger.Range["color"];
}

const read = async (
  client: Synnax,
  parent: ranger.Key,
  after?: ranger.Key,
): Promise<Suggestion> => {
  const [p, unsorted] = await Promise.all([
    client.ranges.retrieve(parent),
    client.ranges.retrieve({ parents: [parent] }),
  ]);
  const children = unsorted.sort(byStart);
  const last = after == null ? children.at(-1) : children.find((c) => c.key === after);
  let template: ranger.Range[] = [];
  if (p.parent != null) {
    const previous = (await client.ranges.retrieve({ parents: [p.parent] }))
      .filter((r) => r.key !== p.key && byStart(r, p) < 0)
      .sort(byStart)
      .at(-1);
    if (previous != null)
      template = (await client.ranges.retrieve({ parents: [previous.key] })).sort(
        byStart,
      );
  }
  const from = (r: ranger.Range): Suggestion => ({ name: r.name, color: r.color });
  if (last == null) return template.length > 0 ? from(template[0]) : { name: "" };
  const i = template.findIndex((t) => t.name === last.name);
  if (i >= 0 && i + 1 < template.length) return from(template[i + 1]);
  const name = increment(last.name);
  return name == null ? { name: "" } : { name, color: last.color };
};

/**
 * The name and color for the next child of `parent`, after the child `after` or after
 * its last child. The parent's previous sibling is the template: if its children ran
 * Chill, Ignition, Mainstage, the child after Ignition is Mainstage. With no match, a
 * trailing number goes up by one and the color stays. A failed read gives no name,
 * since the suggestion only saves typing.
 */
export const suggest = (
  client: Synnax,
  parent: ranger.Key,
  after?: ranger.Key,
): Promise<Suggestion> =>
  read(client, parent, after).catch((error: unknown) => {
    console.error(error);
    return { name: "" };
  });
